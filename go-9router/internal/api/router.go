package api

import (
	"encoding/json"
	"net/http"

	"go-9router/internal/api/middleware"
	"go-9router/internal/config"
	"go-9router/internal/db"
	"go-9router/web"
)

func RegisterRoutes(mux *http.ServeMux, cfg *config.Config, chatHandler *ChatHandler, mitmHandler *MitmHandler, settingsHandler *SettingsHandler, logHandler *LogHandler, dbStore *db.Store) http.Handler {
	keysHandler := NewKeysHandler(dbStore)
	providersHandler := NewProvidersHandler(dbStore)
	usageHandler := NewUsageHandler(dbStore)
	combosHandler := NewCombosHandler(dbStore)
	proxyPoolsHandler := NewProxyPoolsHandler(dbStore)
	multiChatHandler := NewMultiProviderChatHandler(dbStore, nil)

	mux.HandleFunc("/health", HandleHealth)

	// Chat completions: try multi-provider first, fall back to Antigravity-only handler
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		// Parse model to decide which handler to use
		// Peek at the model without consuming the body
		if r.Method != http.MethodPost {
			chatHandler.HandleChatCompletions(w, r)
			return
		}

		// Use multi-provider handler — it handles all providers including Antigravity
		// Falls back across connections automatically
		multiChatHandler.HandleChatCompletions(w, r)
	})

	// API Keys CRUD
	mux.HandleFunc("/api/keys/", keysHandler.HandleKeys)
	mux.HandleFunc("/api/keys", keysHandler.HandleKeys)

	// Providers CRUD
	mux.HandleFunc("/api/providers/", providersHandler.HandleProviders)
	mux.HandleFunc("/api/providers", providersHandler.HandleProviders)

	// Usage Analytics
	mux.HandleFunc("/api/usage/", usageHandler.HandleUsage)
	mux.HandleFunc("/api/usage", usageHandler.HandleUsage)

	// Combos CRUD
	mux.HandleFunc("/api/combos/", combosHandler.HandleCombos)
	mux.HandleFunc("/api/combos", combosHandler.HandleCombos)

	// Proxy Pools CRUD
	mux.HandleFunc("/api/proxy-pools/", proxyPoolsHandler.HandleProxyPools)
	mux.HandleFunc("/api/proxy-pools", proxyPoolsHandler.HandleProxyPools)

	// MITM Control APIs (Ported from Node endpoints)
	mux.HandleFunc("/api/cli-tools/antigravity-mitm", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			mitmHandler.HandleStatus(w, r)
		} else if r.Method == http.MethodPost {
			mitmHandler.HandleControl(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// MITM Alias Mappings APIs (Dynamic aliases)
	mux.HandleFunc("/api/cli-tools/antigravity-mitm/alias", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			mitmHandler.HandleGetAliases(w, r)
		} else if r.Method == http.MethodPut || r.Method == http.MethodPost {
			mitmHandler.HandlePutAliases(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Backup & Restore Database Settings APIs
	mux.HandleFunc("/api/settings/database", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			settingsHandler.HandleExportDatabase(w, r)
		} else if r.Method == http.MethodPost {
			settingsHandler.HandleImportDatabase(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Settings API — GET returns settings, PATCH merges partial updates
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			payload := dbStore.GetPayload()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload.Settings)
		case http.MethodPatch:
			var patch map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"Invalid JSON"}`))
				return
			}
			if err := dbStore.PatchSettings(patch); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"Failed to save settings"}`))
				return
			}
			payload := dbStore.GetPayload()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload.Settings)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Auth status stub — returns no-auth-required for now (auth will be added in Phase 6)
	mux.HandleFunc("/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"requireLogin":false,"authMode":"none","hasPassword":false}`))
	})

	// Version stub — no update mechanism in Go binary, always returns no update
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"0.5.0","hasUpdate":false}`))
	})

	// Console Log Buffer APIs
	mux.HandleFunc("/api/translator/console-logs/stream", logHandler.HandleStream)
	mux.HandleFunc("/api/translator/console-logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodDelete {
			logHandler.HandleLogs(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// CLI Tools status stub — returns empty statuses (real implementation requires filesystem probing)
	mux.HandleFunc("/api/cli-tools/all-statuses", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	// Model aliases CRUD
	mux.HandleFunc("/api/models/alias", func(w http.ResponseWriter, r *http.Request) {
		payload := dbStore.GetPayload()
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"aliases": payload.ModelAliases})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			alias, _ := body["alias"].(string)
			model, _ := body["model"].(string)
			if alias != "" && model != "" {
				dbStore.Lock()
				p := dbStore.GetPayloadUnsafe()
				p.ModelAliases[alias] = model
				dbStore.Unlock()
				_ = dbStore.Save()
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true}`))
		case http.MethodDelete:
			alias := r.URL.Query().Get("alias")
			if alias != "" {
				dbStore.Lock()
				p := dbStore.GetPayloadUnsafe()
				delete(p.ModelAliases, alias)
				dbStore.Unlock()
				_ = dbStore.Save()
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Disabled models stub
	mux.HandleFunc("/api/models/disabled", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"disabled":[]}`))
		case http.MethodPost, http.MethodDelete:
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Model test stub
	mux.HandleFunc("/api/models/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true}`))
	})

	// Provider nodes stub (for compatible providers)
	mux.HandleFunc("/api/provider-nodes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"nodes":[]}`))
		case http.MethodPost:
			_, _ = w.Write([]byte(`{"node":{}}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Batch test providers stub
	mux.HandleFunc("/api/providers/test-batch", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[],"summary":{"total":0,"passed":0,"failed":0}}`))
	})

	// Serve Standalone Dashboard Web UI (SPA with fallback to index.html)
	mux.Handle("/", web.SPAHandler())

	// Wrap with logging & auth middlewares
	var handler http.Handler = mux
	handler = middleware.Auth(cfg, dbStore, handler)
	handler = middleware.Logger(handler)

	return handler
}
