package api

import (
	"net/http"
	"strings"

	"go-9router/internal/db"
)

type UsageHandler struct {
	Store *db.Store
}

func NewUsageHandler(store *db.Store) *UsageHandler {
	return &UsageHandler{Store: store}
}

// HandleUsage routes /api/usage/*
func (h *UsageHandler) HandleUsage(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/usage")
	path = strings.TrimPrefix(path, "/")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	switch path {
	case "", "stats":
		h.handleStats(w, r)
	case "chart":
		h.handleChart(w, r)
	case "history":
		h.handleStats(w, r) // history = stats with period=all
	case "providers":
		h.handleProviders(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
	}
}

// GET /api/usage/stats — aggregate usage from all connections
func (h *UsageHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	payload := h.Store.GetPayload()

	var totalRequests, totalPromptTokens, totalCompletionTokens float64
	byProvider := make(map[string]map[string]interface{})
	byModel := make(map[string]map[string]interface{})

	for _, conn := range payload.ProviderConnections {
		if conn.Data == nil {
			continue
		}

		// Read lifetime totals
		reqLife := toFloat(conn.Data["requestsLifetime"])
		promptLife := toFloat(conn.Data["promptTokensLifetime"])
		complLife := toFloat(conn.Data["completionTokensLifetime"])

		totalRequests += reqLife
		totalPromptTokens += promptLife
		totalCompletionTokens += complLife

		// Aggregate by provider
		provStats, ok := byProvider[conn.Provider]
		if !ok {
			provStats = map[string]interface{}{
				"requests":         0.0,
				"promptTokens":     0.0,
				"completionTokens": 0.0,
				"cost":             0.0,
			}
		}
		provStats["requests"] = toFloat(provStats["requests"]) + reqLife
		provStats["promptTokens"] = toFloat(provStats["promptTokens"]) + promptLife
		provStats["completionTokens"] = toFloat(provStats["completionTokens"]) + complLife
		byProvider[conn.Provider] = provStats

		// Aggregate by model (from usageByModel_* keys)
		for key, val := range conn.Data {
			if !strings.HasPrefix(key, "usageByModel_") {
				continue
			}
			modelName := strings.TrimPrefix(key, "usageByModel_")
			mStats, isMap := val.(map[string]interface{})
			if !isMap {
				continue
			}

			displayKey := modelName + " (" + conn.Provider + ")"
			existing, ok := byModel[displayKey]
			if !ok {
				existing = map[string]interface{}{
					"requests":         0.0,
					"promptTokens":     0.0,
					"completionTokens": 0.0,
					"cost":             0.0,
					"rawModel":         modelName,
					"provider":         conn.Provider,
				}
			}
			existing["requests"] = toFloat(existing["requests"]) + toFloat(mStats["requests"])
			existing["promptTokens"] = toFloat(existing["promptTokens"]) + toFloat(mStats["promptTokens"])
			existing["completionTokens"] = toFloat(existing["completionTokens"]) + toFloat(mStats["completionTokens"])
			byModel[displayKey] = existing
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"totalRequests":         totalRequests,
		"totalPromptTokens":    totalPromptTokens,
		"totalCompletionTokens": totalCompletionTokens,
		"totalCost":            0, // Cost tracking not implemented in Go backend yet
		"byProvider":           byProvider,
		"byModel":              byModel,
		"byAccount":            map[string]interface{}{},
		"byApiKey":             map[string]interface{}{},
		"byEndpoint":           map[string]interface{}{},
		"last10Minutes":        make([]map[string]interface{}, 0),
		"activeRequests":       make([]interface{}, 0),
		"recentRequests":       make([]interface{}, 0),
		"pending":              map[string]interface{}{"byModel": map[string]interface{}{}, "byAccount": map[string]interface{}{}},
		"errorProvider":        "",
	})
}

// GET /api/usage/chart — chart data as array of {label, tokens, cost}
func (h *UsageHandler) handleChart(w http.ResponseWriter, r *http.Request) {
	// Simplified: return aggregate data as a single bucket per provider
	// Full time-series bucketing requires request-level logging (not yet in Go)
	payload := h.Store.GetPayload()

	buckets := make([]map[string]interface{}, 0)

	for _, conn := range payload.ProviderConnections {
		if conn.Data == nil {
			continue
		}
		promptLife := toFloat(conn.Data["promptTokensLifetime"])
		complLife := toFloat(conn.Data["completionTokensLifetime"])
		tokens := promptLife + complLife
		if tokens > 0 {
			buckets = append(buckets, map[string]interface{}{
				"label":  conn.Name,
				"tokens": tokens,
				"cost":   0,
			})
		}
	}

	// If no data, return a single empty bucket
	if len(buckets) == 0 {
		buckets = append(buckets, map[string]interface{}{
			"label":  "No data",
			"tokens": 0,
			"cost":   0,
		})
	}

	writeJSON(w, http.StatusOK, buckets)
}

// GET /api/usage/providers — active providers with usage
func (h *UsageHandler) handleProviders(w http.ResponseWriter, r *http.Request) {
	payload := h.Store.GetPayload()

	providers := make([]map[string]interface{}, 0)
	seen := make(map[string]bool)

	for _, conn := range payload.ProviderConnections {
		if !conn.IsActive || seen[conn.Provider] {
			continue
		}
		seen[conn.Provider] = true
		providers = append(providers, map[string]interface{}{
			"id":       conn.Provider,
			"provider": conn.Provider,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"providers": providers,
	})
}

func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}
