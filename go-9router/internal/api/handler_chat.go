package api

import (
	"encoding/json"
	"log"
	"net/http"

	"go-9router/internal/provider/antigravity"
	"go-9router/internal/sse"
	"go-9router/internal/translator"
)

type ChatHandler struct {
	Executor *antigravity.Executor
}

func NewChatHandler(exec *antigravity.Executor) *ChatHandler {
	return &ChatHandler{
		Executor: exec,
	}
}

func (h *ChatHandler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var openaiReq translator.OpenAIRequest
	if err := json.NewDecoder(r.Body).Decode(&openaiReq); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "Invalid JSON body"}`))
		return
	}

	projectID := antigravity.GenerateProjectID()
	sessionID := r.Header.Get("X-Machine-Session-Id")
	if sessionID == "" {
		sessionID = "go-9router-sess-" + projectID
	}

	// Translate request
	agReq, toolNameMap, err := translator.OpenAIRequestToAntigravity(&openaiReq, projectID, sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Failed to translate request: ` + err.Error() + `"}`))
		return
	}

	state := translator.NewTranslateState(toolNameMap)

	// Execute via provider
	resp, err := h.Executor.Execute(r.Context(), agReq, openaiReq.Stream)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error": "Upstream error: ` + err.Error() + `"}`))
		return
	}

	// Forward stream or json response
	if openaiReq.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		err = sse.ProcessSSEStream(r.Context(), resp.Body, w, state)
		if err != nil {
			log.Printf("SSE processing failed: %v", err)
		}

		// Increment usage for streams if usage was captured
		if state.Usage != nil && h.Executor.DBStore != nil && h.Executor.ConnectionID != "" {
			h.Executor.DBStore.IncrementUsage(h.Executor.ConnectionID, agReq.Model, state.Usage.PromptTokens, state.Usage.CompletionTokens)
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		resBytes, err := sse.ProcessJSONResponse(resp.Body, state)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "Failed to translate response: ` + err.Error() + `"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resBytes)

		// Increment usage for non-streaming requests
		if state.Usage != nil && h.Executor.DBStore != nil && h.Executor.ConnectionID != "" {
			h.Executor.DBStore.IncrementUsage(h.Executor.ConnectionID, agReq.Model, state.Usage.PromptTokens, state.Usage.CompletionTokens)
		}
	}
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status": "ok", "provider": "antigravity"}`))
}
