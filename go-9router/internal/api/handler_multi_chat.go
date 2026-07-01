package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"go-9router/internal/db"
	"go-9router/internal/executor"
)

// MultiProviderChatHandler handles /v1/chat/completions for ALL providers
// using the executor package for format translation and upstream execution.
type MultiProviderChatHandler struct {
	Store      *db.Store
	HTTPClient *http.Client
}

func NewMultiProviderChatHandler(store *db.Store, httpClient *http.Client) *MultiProviderChatHandler {
	return &MultiProviderChatHandler{
		Store:      store,
		HTTPClient: httpClient,
	}
}

func (h *MultiProviderChatHandler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to read request body"})
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON body"})
		return
	}

	modelInput, _ := body["model"].(string)
	if modelInput == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "model is required"})
		return
	}

	stream := false
	if s, ok := body["stream"].(bool); ok {
		stream = s
	}

	// Resolve model → provider + connections
	provider, model, connections, err := executor.ResolveModel(modelInput, h.Store)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Override model in body
	body["model"] = model

	// Try connections in priority order (fallback chain)
	var lastErr error
	for _, conn := range connections {
		creds := executor.GetCredentials(conn)
		if creds.APIKey == "" && creds.AccessToken == "" {
			continue // Skip connections without credentials
		}

		result, execErr := executor.Execute(r.Context(), provider, model, body, stream, creds, h.HTTPClient)
		if execErr != nil {
			lastErr = execErr
			log.Printf("[MULTI] Connection %s (%s) failed: %v", conn.Name, conn.ID, execErr)
			continue
		}

		// Check for upstream error responses
		if result.Response.StatusCode >= 400 {
			respBody, _ := io.ReadAll(result.Response.Body)
			result.Response.Body.Close()
			lastErr = fmt.Errorf("upstream %s returned %d: %s", provider, result.Response.StatusCode, string(respBody))
			log.Printf("[MULTI] Connection %s (%s) upstream error: %d", conn.Name, conn.ID, result.Response.StatusCode)

			// Don't retry on 4xx client errors (except 429)
			if result.Response.StatusCode != 429 && result.Response.StatusCode < 500 {
				writeJSON(w, result.Response.StatusCode, map[string]string{"error": string(respBody)})
				return
			}
			continue
		}

		targetFormat := executor.GetProviderFormat(provider)

		if stream {
			// Stream SSE response, translating chunks from target format → OpenAI
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no")
			w.WriteHeader(http.StatusOK)

			flusher, _ := w.(http.Flusher)

			streamErr := executor.StreamSSE(result.Response, func(data string) error {
				translated := executor.TranslateResponseChunk(targetFormat, data, model)
				if translated == "" {
					return nil // Skip empty translations
				}
				if translated == "[DONE]" {
					fmt.Fprint(w, "data: [DONE]\n\n")
					if flusher != nil {
						flusher.Flush()
					}
					return nil
				}
				fmt.Fprintf(w, "data: %s\n\n", translated)
				if flusher != nil {
					flusher.Flush()
				}
				return nil
			})

			if streamErr != nil {
				log.Printf("[MULTI] SSE stream error: %v", streamErr)
			}

			// Track usage
			h.Store.IncrementUsage(conn.ID, model, 0, 0)
			return
		}

		// Non-streaming: read full response and translate
		respBody, readErr := io.ReadAll(result.Response.Body)
		result.Response.Body.Close()
		if readErr != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Failed to read upstream response"})
			return
		}

		translated, transErr := executor.TranslateNonStreamingResponse(targetFormat, respBody, model)
		if transErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to translate response"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(translated)

		// Track usage
		h.Store.IncrementUsage(conn.ID, model, 0, 0)
		return
	}

	// All connections failed
	errMsg := "All connections failed"
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": errMsg})
}
