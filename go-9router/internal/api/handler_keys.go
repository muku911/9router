package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"go-9router/internal/db"
)

type KeysHandler struct {
	Store *db.Store
}

func NewKeysHandler(store *db.Store) *KeysHandler {
	return &KeysHandler{Store: store}
}

// GET /api/keys — list all API keys
func (h *KeysHandler) HandleListKeys(w http.ResponseWriter, r *http.Request) {
	payload := h.Store.GetPayload()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"keys": payload.ApiKeys,
	})
}

// POST /api/keys — create a new API key
func (h *KeysHandler) HandleCreateKey(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to read request body"})
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(bodyBytes, &body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Name is required"})
		return
	}

	id := uuid.New().String()
	key := fmt.Sprintf("9r-%s", uuid.New().String())
	now := time.Now().UTC().Format(time.RFC3339)

	newKey := map[string]interface{}{
		"id":        id,
		"name":      body.Name,
		"key":       key,
		"isActive":  true,
		"createdAt": now,
		"updatedAt": now,
	}

	h.Store.Lock()
	payload := h.Store.GetPayloadUnsafe()
	payload.ApiKeys = append(payload.ApiKeys, newKey)
	h.Store.Unlock()

	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save after creating key: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"key":  key,
		"name": body.Name,
		"id":   id,
	})
}

// GET /api/keys/{id} — get a single key
func (h *KeysHandler) HandleGetKey(w http.ResponseWriter, r *http.Request, id string) {
	payload := h.Store.GetPayload()
	for _, k := range payload.ApiKeys {
		if keyID, _ := k["id"].(string); keyID == id {
			writeJSON(w, http.StatusOK, map[string]interface{}{"key": k})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Key not found"})
}

// PUT /api/keys/{id} — update a key (e.g. toggle isActive)
func (h *KeysHandler) HandleUpdateKey(w http.ResponseWriter, r *http.Request, id string) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to read request body"})
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	h.Store.Lock()
	payload := h.Store.GetPayloadUnsafe()
	found := false
	for i, k := range payload.ApiKeys {
		if keyID, _ := k["id"].(string); keyID == id {
			// Merge allowed fields
			if isActive, ok := body["isActive"]; ok {
				payload.ApiKeys[i]["isActive"] = isActive
			}
			if name, ok := body["name"]; ok {
				payload.ApiKeys[i]["name"] = name
			}
			payload.ApiKeys[i]["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
			found = true
			break
		}
	}
	h.Store.Unlock()

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Key not found"})
		return
	}

	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save after updating key: %v\n", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"success": "true"})
}

// DELETE /api/keys/{id} — delete a key
func (h *KeysHandler) HandleDeleteKey(w http.ResponseWriter, r *http.Request, id string) {
	h.Store.Lock()
	payload := h.Store.GetPayloadUnsafe()
	found := false
	newKeys := make([]map[string]interface{}, 0, len(payload.ApiKeys))
	for _, k := range payload.ApiKeys {
		if keyID, _ := k["id"].(string); keyID == id {
			found = true
			continue
		}
		newKeys = append(newKeys, k)
	}
	payload.ApiKeys = newKeys
	h.Store.Unlock()

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Key not found"})
		return
	}

	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save after deleting key: %v\n", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Key deleted successfully"})
}

// HandleKeys is the combined router for /api/keys and /api/keys/{id}
func (h *KeysHandler) HandleKeys(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/keys")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		// /api/keys
		switch r.Method {
		case http.MethodGet:
			h.HandleListKeys(w, r)
		case http.MethodPost:
			h.HandleCreateKey(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/keys/{id}
	id := path
	switch r.Method {
	case http.MethodGet:
		h.HandleGetKey(w, r, id)
	case http.MethodPut:
		h.HandleUpdateKey(w, r, id)
	case http.MethodDelete:
		h.HandleDeleteKey(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
