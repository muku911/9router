package middleware

import (
	"log"
	"net/http"
	"time"

	"go-9router/internal/config"
	"go-9router/internal/db"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s took %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Check if raw token matches any active API Key in database payload
func validateDBApiKey(dbStore *db.Store, keyToCheck string) bool {
	if dbStore == nil || keyToCheck == "" {
		return false
	}

	payload := dbStore.GetPayload()
	if payload == nil || len(payload.ApiKeys) == 0 {
		return false
	}

	for _, k := range payload.ApiKeys {
		isActive := true
		if actVal, ok := k["isActive"]; ok {
			isActive = actVal.(bool)
		}
		if keyVal, ok := k["key"]; ok && keyVal.(string) == keyToCheck && isActive {
			return true
		}
	}
	return false
}

func Auth(cfg *config.Config, dbStore *db.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public paths
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Read DB settings: check if API Key is required by settings table
		// settings.requireApiKey or settings.requireLogin
		dbRequireApiKey := false
		if dbStore != nil {
			payload := dbStore.GetPayload()
			if payload.Settings != nil {
				if reqKey, ok := payload.Settings["requireApiKey"]; ok {
					dbRequireApiKey = reqKey.(bool)
				}
			}
		}

		// Auth is disabled if neither Basic Auth nor API Key (both env and DB settings) is configured/required
		if cfg.APIKey == "" && (cfg.Username == "" || cfg.Password == "") && !dbRequireApiKey {
			next.ServeHTTP(w, r)
			return
		}

		isAuthorized := false

		// 1. Check Basic Authentication first if configured
		if cfg.Username != "" && cfg.Password != "" {
			user, pass, ok := r.BasicAuth()
			if ok && user == cfg.Username && pass == cfg.Password {
				isAuthorized = true
			}
		}

		// 2. Check API Key fallback (prioritize Env cfg.APIKey, then database ApiKeys)
		if !isAuthorized {
			authHeader := r.Header.Get("Authorization")
			apiKey := r.Header.Get("x-api-key")

			var keyToCheck string
			if apiKey != "" {
				keyToCheck = apiKey
			} else if authHeader != "" {
				parts := r.Header.Values("Authorization")
				for _, val := range parts {
					if val == "Bearer " {
						continue
					}
					if len(val) > 7 && val[:7] == "Bearer " {
						keyToCheck = val[7:]
						break
					} else {
						keyToCheck = val
						break
					}
				}
			}

			if keyToCheck != "" {
				// A. Check against Env config
				if cfg.APIKey != "" && keyToCheck == cfg.APIKey {
					isAuthorized = true
				}
				// B. Check against DB ApiKeys
				if !isAuthorized && dbStore != nil {
					isAuthorized = validateDBApiKey(dbStore, keyToCheck)
				}
			}
		}

		if !isAuthorized {
			if cfg.Username != "" && cfg.Password != "" {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "Unauthorized"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
