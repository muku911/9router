package api

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"go-9router/internal/db"
)

type CombosHandler struct {
	Store *db.Store
}

func NewCombosHandler(store *db.Store) *CombosHandler {
	return &CombosHandler{Store: store}
}

func (h *CombosHandler) HandleCombos(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/combos")
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

	id := path
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

func (h *CombosHandler) handleList(w http.ResponseWriter, r *http.Request) {
	payload := h.Store.GetPayload()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"combos": payload.Combos,
	})
}

func (h *CombosHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	combo := make(map[string]interface{})
	if err := jsonUnmarshal(bodyBytes, &combo); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	combo["id"] = uuid.New().String()
	combo["createdAt"] = time.Now().UTC().Format(time.RFC3339)

	h.Store.Lock()
	p := h.Store.GetPayloadUnsafe()
	p.Combos = append(p.Combos, combo)
	h.Store.Unlock()

	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save combo: %v\n", err)
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"combo": combo})
}

func (h *CombosHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	payload := h.Store.GetPayload()
	for _, c := range payload.Combos {
		if cid, _ := c["id"].(string); cid == id {
			writeJSON(w, http.StatusOK, map[string]interface{}{"combo": c})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Combo not found"})
}

func (h *CombosHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	bodyBytes, _ := io.ReadAll(r.Body)
	patch := make(map[string]interface{})
	if err := jsonUnmarshal(bodyBytes, &patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	h.Store.Lock()
	p := h.Store.GetPayloadUnsafe()
	found := false
	for i, c := range p.Combos {
		if cid, _ := c["id"].(string); cid == id {
			for k, v := range patch {
				p.Combos[i][k] = v
			}
			p.Combos[i]["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
			found = true
			break
		}
	}
	h.Store.Unlock()

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Combo not found"})
		return
	}
	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save combo: %v\n", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"success": "true"})
}

func (h *CombosHandler) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	h.Store.Lock()
	p := h.Store.GetPayloadUnsafe()
	found := false
	newCombos := make([]map[string]interface{}, 0)
	for _, c := range p.Combos {
		if cid, _ := c["id"].(string); cid == id {
			found = true
			continue
		}
		newCombos = append(newCombos, c)
	}
	p.Combos = newCombos
	h.Store.Unlock()

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Combo not found"})
		return
	}
	if err := h.Store.Save(); err != nil {
		log.Printf("Failed to save combo: %v\n", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Combo deleted"})
}
