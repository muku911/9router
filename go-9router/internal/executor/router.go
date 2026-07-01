package executor

import (
	"fmt"
	"strings"

	"go-9router/internal/db"
)

// ResolveModel takes a model string (possibly with provider prefix like "openai/gpt-4o")
// and returns the provider ID, the clean model name, and a list of matching connections.
// It also checks combos and model aliases.
func ResolveModel(modelInput string, store *db.Store) (provider string, model string, connections []db.ProviderConnection, err error) {
	payload := store.GetPayload()

	// Check if model is a combo name
	for _, combo := range payload.Combos {
		name, _ := combo["name"].(string)
		if name == modelInput {
			// Return the first model in the combo as the primary
			models, _ := combo["models"].([]interface{})
			if len(models) > 0 {
				firstModel, _ := models[0].(string)
				if firstModel != "" {
					return ResolveModel(firstModel, store)
				}
			}
		}
	}

	// Check model aliases (map of alias → target model)
	for aliasName, target := range payload.ModelAliases {
		if aliasName == modelInput {
			if targetStr, ok := target.(string); ok && targetStr != "" {
				return ResolveModel(targetStr, store)
			}
		}
	}

	// Parse provider/model format
	provider, model = parseModelString(modelInput)

	// Find matching connections
	connections = findConnections(provider, payload)

	if len(connections) == 0 {
		// Try without provider prefix — search all connections
		if provider != "" {
			allConns := findConnections("", payload)
			if len(allConns) > 0 {
				connections = allConns
				// Use the first connection's provider
				provider = connections[0].Provider
			}
		}
	}

	if len(connections) == 0 {
		return "", model, nil, fmt.Errorf("no active connections found for model %q", modelInput)
	}

	return provider, model, connections, nil
}

// parseModelString splits "provider/model" into (provider, model).
// If no slash, returns ("", model).
func parseModelString(input string) (string, string) {
	// Handle provider alias prefixes like "openai/gpt-4o"
	parts := strings.SplitN(input, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", input
}

// findConnections returns active connections for a provider.
// If provider is empty, returns all active connections.
func findConnections(provider string, payload *db.DatabasePayload) []db.ProviderConnection {
	var result []db.ProviderConnection
	for _, conn := range payload.ProviderConnections {
		if !conn.IsActive {
			continue
		}
		if provider != "" {
			// Match by provider ID or alias
			if conn.Provider != provider {
				cfg := GetProviderConfig(conn.Provider)
				_ = cfg // Could check aliases here
				continue
			}
		}
		result = append(result, conn)
	}

	// Sort by priority
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Priority < result[i].Priority {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// GetCredentials extracts auth credentials from a connection.
func GetCredentials(conn db.ProviderConnection) Credentials {
	creds := Credentials{}

	// Check explicit fields first
	if conn.AccessToken != "" {
		creds.AccessToken = conn.AccessToken
	}

	// Check Data map
	if conn.Data != nil {
		if apiKey, ok := conn.Data["apiKey"].(string); ok && apiKey != "" {
			creds.APIKey = apiKey
		}
		if accessToken, ok := conn.Data["accessToken"].(string); ok && accessToken != "" {
			creds.AccessToken = accessToken
		}
	}

	return creds
}
