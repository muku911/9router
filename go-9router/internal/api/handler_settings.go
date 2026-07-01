package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"go-9router/internal/db"
)

type SettingsHandler struct {
	Store    *db.Store
	OnReload func()
}

func NewSettingsHandler(store *db.Store, onReload func()) *SettingsHandler {
	return &SettingsHandler{
		Store:    store,
		OnReload: onReload,
	}
}

func (h *SettingsHandler) HandleExportDatabase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	payload := h.Store.GetPayload()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *SettingsHandler) HandleImportDatabase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "Failed to read request body"}`))
		return
	}

	if err := h.Store.Import(bodyBytes); err != nil {
		log.Printf("Import failed: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "` + err.Error() + `"}`))
		return
	}

	// Trigger dynamic reloading of proxies and credentials
	if h.OnReload != nil {
		go h.OnReload()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success": true}`))
}
