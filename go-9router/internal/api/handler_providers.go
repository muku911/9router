package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"go-9router/internal/db"
)

type ProvidersHandler struct {
	Store *db.Store
}

func NewProvidersHandler(store *db.Store) *ProvidersHandler {
	return &ProvidersHandler{Store: store}
}

// HandleProviders routes /api/providers and /api/providers/{id}
func (h *ProvidersHandler) HandleProviders(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/providers")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	// Check for sub-routes: /api/providers/{id}/test
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]

	if len(parts) == 2 && parts[1] == "test" {
		if r.Method == http.MethodPost {
			h.handleTest(w, r, id)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/providers/{id}
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, id)
	case http.MethodPut:
		h.handleUpdate(w, r, id)
	case http.MethodDelete:
		h.handleDelete(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// GET /api/providers — list all connections (sensitive fields stripped)
func (h *ProvidersHandler) handleList(w http.ResponseWriter, r *http.Request) {
	payload := h.Store.GetPayload()
	connections := make([]map[string]interface{}, 0, len(payload.ProviderConnections))

	for _, conn := range payload.ProviderConnections {
		connections = append(connections, stripSecrets(connToMap(conn)))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connections": connections,
	})
}

// POST /api/providers — create a new connection
func (h *ProvidersHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to read request body"})
		return
	}

	var body struct {
		Provider     string `json:"provider"`
		AuthType     string `json:"authType"`
		Name         string `json:"name"`
		DisplayName  string `json:"displayName"`
		Email        string `json:"email"`
		ApiKey       string `json:"apiKey"`
		RefreshToken string `json:"refreshToken"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
		IsActive     *bool  `json:"isActive"`
		Priority     *int   `json:"priority"`
	}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	if body.Provider == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Provider is required"})
		return
	}

	name := body.Name
	if name == "" {
		name = body.DisplayName
	}
	if name == "" {
		name = body.Provider
	}

	authType := body.AuthType
	if authType == "" {
		if body.ApiKey != "" {
			authType = "apikey"
		} else if body.RefreshToken != "" {
			authType = "oauth"
		} else {
			authType = "apikey"
		}
	}

	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	// Compute priority
	priority := 1
	if body.Priority != nil {
		priority = *body.Priority
	} else {
		payload := h.Store.GetPayload()
		maxPriority := 0
		for _, c := range payload.ProviderConnections {
			if c.Provider == body.Provider && c.Priority > maxPriority {
				maxPriority = c.Priority
			}
		}
		priority = maxPriority + 1
	}

	now := time.Now().UTC().Format(time.RFC3339)
	conn := db.ProviderConnection{
		ID:           uuid.New().String(),
		Provider:     body.Provider,
		AuthType:     authType,
		Name:         name,
		Email:        body.Email,
		Priority:     priority,
		IsActive:     isActive,
		RefreshToken: body.RefreshToken,
		ClientID:     body.ClientID,
		ClientSecret: body.ClientSecret,
		CreatedAt:    now,
		UpdatedAt:    now,
		Data:         make(map[string]interface{}),
	}

	// Store apiKey in Data map (matches how Go DB stores it)
	if body.ApiKey != "" {
		conn.Data["apiKey"] = body.ApiKey
	}
	if body.DisplayName != "" {
		conn.Data["displayName"] = body.DisplayName
	}

	// Parse any extra fields from the raw body into Data
	var rawBody map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &rawBody)
	for _, k := range []string{"testStatus", "defaultModel", "globalPriority", "providerSpecificData",
		"connectionProxyEnabled", "connectionProxyUrl", "connectionNoProxy", "proxyPoolId"} {
		if v, ok := rawBody[k]; ok {
			conn.Data[k] = v
		}
	}

	h.Store.Lock()
	p := h.Store.GetPayloadUnsafe()
	p.ProviderConnections = append(p.ProviderConnections, conn)
	h.Store.Unlock()

	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save after creating provider: %v\n", err)
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"connection": stripSecrets(connToMap(conn)),
	})
}

// GET /api/providers/{id}
func (h *ProvidersHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	payload := h.Store.GetPayload()
	for _, conn := range payload.ProviderConnections {
		if conn.ID == id {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"connection": stripSecrets(connToMap(conn)),
			})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Connection not found"})
}

// PUT /api/providers/{id}
func (h *ProvidersHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to read request body"})
		return
	}

	var patch map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	h.Store.Lock()
	p := h.Store.GetPayloadUnsafe()
	found := false
	var updated db.ProviderConnection
	for i, conn := range p.ProviderConnections {
		if conn.ID == id {
			// Apply allowed top-level fields
			if v, ok := patch["name"].(string); ok {
				p.ProviderConnections[i].Name = v
			}
			if v, ok := patch["isActive"].(bool); ok {
				p.ProviderConnections[i].IsActive = v
			}
			if v, ok := patch["priority"]; ok {
				if pf, ok2 := v.(float64); ok2 {
					p.ProviderConnections[i].Priority = int(pf)
				}
			}
			if v, ok := patch["email"].(string); ok {
				p.ProviderConnections[i].Email = v
			}

			// Merge data-level fields
			if p.ProviderConnections[i].Data == nil {
				p.ProviderConnections[i].Data = make(map[string]interface{})
			}
			for _, k := range []string{"apiKey", "displayName", "testStatus", "lastError", "lastErrorAt",
				"defaultModel", "globalPriority", "providerSpecificData",
				"connectionProxyEnabled", "connectionProxyUrl", "connectionNoProxy", "proxyPoolId"} {
				if v, ok := patch[k]; ok {
					p.ProviderConnections[i].Data[k] = v
				}
			}

			p.ProviderConnections[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			updated = p.ProviderConnections[i]
			found = true
			break
		}
	}
	h.Store.Unlock()

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Connection not found"})
		return
	}

	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save after updating provider: %v\n", err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connection": stripSecrets(connToMap(updated)),
	})
}

// DELETE /api/providers/{id}
func (h *ProvidersHandler) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	h.Store.Lock()
	p := h.Store.GetPayloadUnsafe()
	found := false
	newConns := make([]db.ProviderConnection, 0, len(p.ProviderConnections))
	for _, c := range p.ProviderConnections {
		if c.ID == id {
			found = true
			continue
		}
		newConns = append(newConns, c)
	}
	p.ProviderConnections = newConns
	h.Store.Unlock()

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Connection not found"})
		return
	}

	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save after deleting provider: %v\n", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Connection deleted successfully"})
}

// POST /api/providers/{id}/test — stub (real implementation needs upstream probing)
func (h *ProvidersHandler) handleTest(w http.ResponseWriter, r *http.Request, id string) {
	payload := h.Store.GetPayload()
	for _, conn := range payload.ProviderConnections {
		if conn.ID == id {
			// Stub: mark as tested
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"valid":     true,
				"error":     nil,
				"refreshed": false,
			})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Connection not found"})
}

// --- helpers ---

func connToMap(c db.ProviderConnection) map[string]interface{} {
	m := map[string]interface{}{
		"id":        c.ID,
		"provider":  c.Provider,
		"authType":  c.AuthType,
		"name":      c.Name,
		"email":     c.Email,
		"priority":  c.Priority,
		"isActive":  c.IsActive,
		"createdAt": c.CreatedAt,
		"updatedAt": c.UpdatedAt,
	}
	// Flatten data into the map
	for k, v := range c.Data {
		m[k] = v
	}
	// Also include explicit fields if set
	if c.RefreshToken != "" {
		m["refreshToken"] = c.RefreshToken
	}
	if c.ClientID != "" {
		m["clientId"] = c.ClientID
	}
	if c.ClientSecret != "" {
		m["clientSecret"] = c.ClientSecret
	}
	if c.AccessToken != "" {
		m["accessToken"] = c.AccessToken
	}
	if c.Locks != nil {
		m["locks"] = c.Locks
	}
	return m
}

func stripSecrets(m map[string]interface{}) map[string]interface{} {
	for _, k := range []string{"apiKey", "accessToken", "refreshToken", "clientId", "clientSecret", "idToken"} {
		delete(m, k)
	}
	return m
}
