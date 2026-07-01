package api

import (
	"encoding/json"
	"net/http"

	"go-9router/internal/db"
	"go-9router/internal/mitm"
)

type MitmHandler struct {
	Server  *mitm.Server
	DBStore *db.Store
}

func NewMitmHandler(server *mitm.Server, dbStore *db.Store) *MitmHandler {
	return &MitmHandler{
		Server:  server,
		DBStore: dbStore,
	}
}

type MitmStatus struct {
	Running bool   `json:"running"`
	Dns     bool   `json:"dns"`
	CA      string `json:"ca_path"`
}

func (h *MitmHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	status := MitmStatus{
		Running: h.Server.IsRunning(),
		Dns:     mitm.HasDNSEntry("cloudcode-pa.googleapis.com"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(status)
}

type MitmControlRequest struct {
	Action   string `json:"action"` // "start", "stop", "dns_enable", "dns_disable"
	Password string `json:"password,omitempty"`
}

func (h *MitmHandler) HandleControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req MitmControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "Invalid body"}`))
		return
	}

	var err error
	switch req.Action {
	case "start":
		err = h.Server.Start()
	case "stop":
		err = h.Server.Stop()
	case "dns_enable":
		err = mitm.AddDNSEntries(req.Password)
	case "dns_disable":
		err = mitm.RemoveDNSEntries(req.Password)
	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "Unknown action"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "` + err.Error() + `"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success": true}`))
}

func (h *MitmHandler) HandleGetAliases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tool := r.URL.Query().Get("tool")
	if tool == "" {
		tool = "antigravity"
	}

	payload := h.DBStore.GetPayload()
	aliases := make(map[string]interface{})

	if payload.MitmAlias != nil {
		if val, exists := payload.MitmAlias[tool]; exists {
			if m, ok := val.(map[string]interface{}); ok {
				aliases = m
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"aliases": aliases,
	})
}

type MitmAliasUpdateRequest struct {
	Tool     string                 `json:"tool"`
	Mappings map[string]interface{} `json:"mappings"`
}

func (h *MitmHandler) HandlePutAliases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req MitmAliasUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "Invalid body"}`))
		return
	}

	if req.Tool == "" {
		req.Tool = "antigravity"
	}

	payload := h.DBStore.GetPayload()
	h.DBStore.Lock() // Access Store locks manually for updates
	if payload.MitmAlias == nil {
		payload.MitmAlias = make(map[string]interface{})
	}

	payload.MitmAlias[req.Tool] = req.Mappings
	h.DBStore.Unlock()

	if err := h.DBStore.Save(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Failed to save aliases: ` + err.Error() + `"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"aliases": req.Mappings,
	})
}

