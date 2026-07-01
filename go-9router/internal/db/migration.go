package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// CheckAndMigrate checks if the SQLite database exists and if JSON database does not,
// then performs a one-time migration to the Go JSON format.
func CheckAndMigrate(jsonPath string) error {
	// If json file already exists, no migration is needed
	if _, err := os.Stat(jsonPath); err == nil {
		return nil
	}

	// Determine sqlite file path based on the JSON directory
	dbDir := filepath.Dir(jsonPath)
	sqlitePath := filepath.Join(dbDir, "data.sqlite")

	// If sqlite file does not exist, nothing to migrate
	if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
		return nil
	}

	log.Printf("Found existing Node.js SQLite database at %s. Migrating to Go JSON database format at %s...", sqlitePath, jsonPath)

	payload, err := MigrateSQLiteToPayload(sqlitePath)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Marshal and write to JSON file
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal migrated payload: %w", err)
	}

	// Write JSON atomically
	tmpFile := jsonPath + ".tmp"
	if err := os.WriteFile(tmpFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write migrated JSON: %w", err)
	}

	if err := os.Rename(tmpFile, jsonPath); err != nil {
		return fmt.Errorf("failed to rename migrated JSON file: %w", err)
	}

	// Rename SQLite file to prevent running migration again and preserve it as backup
	migratedPath := sqlitePath + ".migrated"
	if err := os.Rename(sqlitePath, migratedPath); err != nil {
		log.Printf("Warning: Failed to rename migrated SQLite database: %v. Please rename or delete it manually.", err)
	} else {
		log.Printf("Successfully renamed SQLite database to %s", migratedPath)
	}

	log.Printf("Migration completed successfully!")
	return nil
}

// MigrateSQLiteToPayload parses the SQLite database structures and converts them to DatabasePayload
func MigrateSQLiteToPayload(sqlitePath string) (*DatabasePayload, error) {
	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}
	defer db.Close()

	payload := &DatabasePayload{
		Settings:            make(map[string]interface{}),
		ProviderConnections: []ProviderConnection{},
		ProviderNodes:       []map[string]interface{}{},
		ProxyPools:          []ProxyPool{},
		ApiKeys:             []map[string]interface{}{},
		Combos:              []map[string]interface{}{},
		ModelAliases:        make(map[string]interface{}),
		CustomModels:        []map[string]interface{}{},
		MitmAlias:           make(map[string]interface{}),
		Pricing:             make(map[string]interface{}),
	}

	// 1. Settings Table
	if err := migrateSettings(db, payload); err != nil {
		log.Printf("Warning: failed to migrate settings: %v", err)
	}

	// 2. ProviderConnections Table
	if err := migrateProviderConnections(db, payload); err != nil {
		log.Printf("Warning: failed to migrate provider connections: %v", err)
	}

	// 3. ProviderNodes Table
	if err := migrateProviderNodes(db, payload); err != nil {
		log.Printf("Warning: failed to migrate provider nodes: %v", err)
	}

	// 4. ProxyPools Table
	if err := migrateProxyPools(db, payload); err != nil {
		log.Printf("Warning: failed to migrate proxy pools: %v", err)
	}

	// 5. ApiKeys Table
	if err := migrateApiKeys(db, payload); err != nil {
		log.Printf("Warning: failed to migrate api keys: %v", err)
	}

	// 6. Combos Table
	if err := migrateCombos(db, payload); err != nil {
		log.Printf("Warning: failed to migrate combos: %v", err)
	}

	// 7. KV Table
	if err := migrateKV(db, payload); err != nil {
		log.Printf("Warning: failed to migrate kv entries: %v", err)
	}

	return payload, nil
}

func migrateSettings(db *sql.DB, payload *DatabasePayload) error {
	row := db.QueryRow("SELECT data FROM settings WHERE id = 1")
	var dataStr string
	if err := row.Scan(&dataStr); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	if err := json.Unmarshal([]byte(dataStr), &payload.Settings); err != nil {
		return fmt.Errorf("failed to unmarshal settings data: %w", err)
	}
	return nil
}

func migrateProviderConnections(db *sql.DB, payload *DatabasePayload) error {
	rows, err := db.Query("SELECT id, provider, authType, name, email, priority, isActive, data, createdAt, updatedAt FROM providerConnections")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, provider, authType, createdAt, updatedAt string
		var name, email sql.NullString
		var priority sql.NullInt64
		var isActiveInt int
		var dataStr string

		err := rows.Scan(&id, &provider, &authType, &name, &email, &priority, &isActiveInt, &dataStr, &createdAt, &updatedAt)
		if err != nil {
			return err
		}

		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &extraData); err != nil {
			extraData = make(map[string]interface{})
		}

		conn := ProviderConnection{
			ID:        id,
			Provider:  provider,
			AuthType:  authType,
			IsActive:  isActiveInt == 1,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Locks:     make(map[string]string),
			Data:      make(map[string]interface{}),
		}

		if name.Valid {
			conn.Name = name.String
		}
		if email.Valid {
			conn.Email = email.String
		}
		if priority.Valid {
			conn.Priority = int(priority.Int64)
		}

		for k, v := range extraData {
			switch k {
			case "refreshToken":
				if s, ok := v.(string); ok {
					conn.RefreshToken = s
				}
			case "accessToken":
				if s, ok := v.(string); ok {
					conn.AccessToken = s
				}
			case "clientId":
				if s, ok := v.(string); ok {
					conn.ClientID = s
				}
			case "clientSecret":
				if s, ok := v.(string); ok {
					conn.ClientSecret = s
				}
			default:
				if strings.HasPrefix(k, "modelLock_") || strings.HasPrefix(k, "modelGroupLock_") {
					if s, ok := v.(string); ok {
						conn.Locks[k] = s
					}
				} else {
					conn.Data[k] = v
				}
			}
		}

		payload.ProviderConnections = append(payload.ProviderConnections, conn)
	}

	return nil
}

func migrateProviderNodes(db *sql.DB, payload *DatabasePayload) error {
	rows, err := db.Query("SELECT id, type, name, data, createdAt, updatedAt FROM providerNodes")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var typeCol, nameCol sql.NullString
		var dataStr, createdAt, updatedAt string

		if err := rows.Scan(&id, &typeCol, &nameCol, &dataStr, &createdAt, &updatedAt); err != nil {
			return err
		}

		var node map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &node); err != nil {
			node = make(map[string]interface{})
		}

		node["id"] = id
		if typeCol.Valid {
			node["type"] = typeCol.String
		} else {
			node["type"] = nil
		}
		if nameCol.Valid {
			node["name"] = nameCol.String
		} else {
			node["name"] = nil
		}
		node["createdAt"] = createdAt
		node["updatedAt"] = updatedAt

		payload.ProviderNodes = append(payload.ProviderNodes, node)
	}
	return nil
}

func migrateProxyPools(db *sql.DB, payload *DatabasePayload) error {
	rows, err := db.Query("SELECT id, isActive, testStatus, data, createdAt, updatedAt FROM proxyPools")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, createdAt, updatedAt string
		var isActiveInt int
		var testStatusCol sql.NullString
		var dataStr string

		if err := rows.Scan(&id, &isActiveInt, &testStatusCol, &dataStr, &createdAt, &updatedAt); err != nil {
			return err
		}

		var poolData map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &poolData); err != nil {
			poolData = make(map[string]interface{})
		}

		pool := ProxyPool{
			ID:        id,
			IsActive:  isActiveInt == 1,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Data:      poolData,
		}
		if testStatusCol.Valid {
			pool.TestStatus = testStatusCol.String
		}

		payload.ProxyPools = append(payload.ProxyPools, pool)
	}
	return nil
}

func migrateApiKeys(db *sql.DB, payload *DatabasePayload) error {
	rows, err := db.Query("SELECT id, key, name, machineId, isActive, createdAt FROM apiKeys")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, key, createdAt string
		var name, machineId sql.NullString
		var isActiveInt int

		if err := rows.Scan(&id, &key, &name, &machineId, &isActiveInt, &createdAt); err != nil {
			return err
		}

		apiKey := map[string]interface{}{
			"id":        id,
			"key":       key,
			"isActive":  isActiveInt == 1,
			"createdAt": createdAt,
		}
		if name.Valid {
			apiKey["name"] = name.String
		} else {
			apiKey["name"] = nil
		}
		if machineId.Valid {
			apiKey["machineId"] = machineId.String
		} else {
			apiKey["machineId"] = nil
		}

		payload.ApiKeys = append(payload.ApiKeys, apiKey)
	}
	return nil
}

func migrateCombos(db *sql.DB, payload *DatabasePayload) error {
	rows, err := db.Query("SELECT id, name, kind, models, createdAt, updatedAt FROM combos")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, modelsStr, createdAt, updatedAt string
		var kindCol sql.NullString

		if err := rows.Scan(&id, &name, &kindCol, &modelsStr, &createdAt, &updatedAt); err != nil {
			return err
		}

		var modelsList []interface{}
		if err := json.Unmarshal([]byte(modelsStr), &modelsList); err != nil {
			modelsList = []interface{}{}
		}

		combo := map[string]interface{}{
			"id":        id,
			"name":      name,
			"models":    modelsList,
			"createdAt": createdAt,
			"updatedAt": updatedAt,
		}
		if kindCol.Valid {
			combo["kind"] = kindCol.String
		} else {
			combo["kind"] = nil
		}

		payload.Combos = append(payload.Combos, combo)
	}
	return nil
}

func migrateKV(db *sql.DB, payload *DatabasePayload) error {
	rows, err := db.Query("SELECT scope, key, value FROM kv")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var scope, key, valueStr string
		if err := rows.Scan(&scope, &key, &valueStr); err != nil {
			return err
		}

		var parsedValue interface{}
		if err := json.Unmarshal([]byte(valueStr), &parsedValue); err != nil {
			parsedValue = valueStr
		}

		switch scope {
		case "modelAliases":
			payload.ModelAliases[key] = parsedValue
		case "customModels":
			if valMap, ok := parsedValue.(map[string]interface{}); ok {
				payload.CustomModels = append(payload.CustomModels, valMap)
			} else if valSlice, ok := parsedValue.([]interface{}); ok {
				for _, item := range valSlice {
					if itemMap, ok := item.(map[string]interface{}); ok {
						payload.CustomModels = append(payload.CustomModels, itemMap)
					}
				}
			}
		case "mitmAlias":
			payload.MitmAlias[key] = parsedValue
		case "pricing":
			payload.Pricing[key] = parsedValue
		}
	}
	return nil
}
