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

type ProxyPoolsHandler struct {
	Store *db.Store
}

func NewProxyPoolsHandler(store *db.Store) *ProxyPoolsHandler {
	return &ProxyPoolsHandler{Store: store}
}

func (h *ProxyPoolsHandler) HandleProxyPools(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/proxy-pools")
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

	parts := strings.SplitN(path, "/", 2)
	id := parts[0]

	if len(parts) == 2 && parts[1] == "test" {
		if r.Method == http.MethodPost {
			writeJSON(w, http.StatusOK, map[string]interface{}{"valid": true, "error": nil})
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

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

func (h *ProxyPoolsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	payload := h.Store.GetPayload()
	// Convert to maps for JSON serialization with flat fields
	pools := make([]map[string]interface{}, 0, len(payload.ProxyPools))
	for _, p := range payload.ProxyPools {
		pools = append(pools, poolToMap(p))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"pools": pools})
}

func (h *ProxyPoolsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var body map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	pool := db.ProxyPool{
		ID:        uuid.New().String(),
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
		Data:      make(map[string]interface{}),
	}
	// Store all fields in Data
	for k, v := range body {
		switch k {
		case "isActive":
			if b, ok := v.(bool); ok {
				pool.IsActive = b
			}
		default:
			pool.Data[k] = v
		}
	}

	h.Store.Lock()
	p := h.Store.GetPayloadUnsafe()
	p.ProxyPools = append(p.ProxyPools, pool)
	h.Store.Unlock()

	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save proxy pool: %v\n", err)
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"pool": poolToMap(pool)})
}

func (h *ProxyPoolsHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	payload := h.Store.GetPayload()
	for _, p := range payload.ProxyPools {
		if p.ID == id {
			writeJSON(w, http.StatusOK, map[string]interface{}{"pool": poolToMap(p)})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Pool not found"})
}

func (h *ProxyPoolsHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var patch map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	h.Store.Lock()
	p := h.Store.GetPayloadUnsafe()
	found := false
	for i, pool := range p.ProxyPools {
		if pool.ID == id {
			if v, ok := patch["isActive"].(bool); ok {
				p.ProxyPools[i].IsActive = v
			}
			if p.ProxyPools[i].Data == nil {
				p.ProxyPools[i].Data = make(map[string]interface{})
			}
			for k, v := range patch {
				if k != "isActive" && k != "id" {
					p.ProxyPools[i].Data[k] = v
				}
			}
			p.ProxyPools[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			found = true
			break
		}
	}
	h.Store.Unlock()

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Pool not found"})
		return
	}
	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save proxy pool: %v\n", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"success": "true"})
}

func (h *ProxyPoolsHandler) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	h.Store.Lock()
	p := h.Store.GetPayloadUnsafe()
	found := false
	newPools := make([]db.ProxyPool, 0, len(p.ProxyPools))
	for _, pool := range p.ProxyPools {
		if pool.ID == id {
			found = true
			continue
		}
		newPools = append(newPools, pool)
	}
	p.ProxyPools = newPools
	h.Store.Unlock()

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Pool not found"})
		return
	}
	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save proxy pool: %v\n", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Pool deleted"})
}

func poolToMap(p db.ProxyPool) map[string]interface{} {
	m := map[string]interface{}{
		"id":        p.ID,
		"isActive":  p.IsActive,
		"createdAt": p.CreatedAt,
		"updatedAt": p.UpdatedAt,
	}
	if p.TestStatus != "" {
		m["testStatus"] = p.TestStatus
	}
	for k, v := range p.Data {
		m[k] = v
	}
	return m
}
