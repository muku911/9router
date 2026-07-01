package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go-9router/internal/utils"
)

type LogHandler struct {
	LogBuffer *utils.LogBuffer
}

func NewLogHandler(lb *utils.LogBuffer) *LogHandler {
	return &LogHandler{
		LogBuffer: lb,
	}
}

func (h *LogHandler) HandleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		logs := h.LogBuffer.GetLogs()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"logs":    logs,
		})
		return
	}

	if r.Method == http.MethodDelete {
		h.LogBuffer.ClearLogs()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// HandleStream serves an SSE stream of console log events.
// Events:
//   - init: {type:"init", logs:[...]}   — sent once with all buffered lines
//   - line: {type:"line", line:"..."}   — sent for each new log line
//   - clear: {type:"clear"}             — sent when logs are cleared
//
// Keepalive comment `: ping` sent every 25 seconds.
func (h *LogHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send initial buffered logs
	initLogs := h.LogBuffer.GetLogs()
	initData, _ := json.Marshal(map[string]interface{}{
		"type": "init",
		"logs": initLogs,
	})
	fmt.Fprintf(w, "data: %s\n\n", initData)
	flusher.Flush()

	// Subscribe to new log lines
	ch, cancel := h.LogBuffer.Subscribe()
	defer cancel()

	// Keepalive ticker
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case line := <-ch:
			if line == "\x00CLEAR\x00" {
				clearData, _ := json.Marshal(map[string]string{"type": "clear"})
				fmt.Fprintf(w, "data: %s\n\n", clearData)
			} else {
				lineData, _ := json.Marshal(map[string]interface{}{
					"type": "line",
					"line": line,
				})
				fmt.Fprintf(w, "data: %s\n\n", lineData)
			}
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
