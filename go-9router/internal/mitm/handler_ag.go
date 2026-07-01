package mitm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-9router/internal/provider/antigravity"
	"go-9router/internal/translator"
)

const (
	AntigravityIdeVersion = "1.23.2"
)

type AntigravityHandler struct {
	Executor *antigravity.Executor
}

func NewAntigravityHandler(exec *antigravity.Executor) *AntigravityHandler {
	return &AntigravityHandler{
		Executor: exec,
	}
}

type Metadata struct {
	IdeName    string `json:"ideName,omitempty"`
	IdeType    string `json:"ideType,omitempty"`
	IdeVersion string `json:"ideVersion,omitempty"`
}

type AntigravityRawPayload struct {
	Model    string    `json:"model,omitempty"`
	Metadata *Metadata `json:"metadata,omitempty"`
}

func shouldRewriteMetadata(m *Metadata) bool {
	if m == nil {
		return false
	}
	if strings.ToLower(m.IdeName) == "antigravity" {
		return true
	}
	if strings.ToUpper(m.IdeType) == "ANTIGRAVITY" {
		return true
	}
	return m.IdeVersion != ""
}

func rewriteUserAgent(userAgent, version string) string {
	if !strings.Contains(userAgent, "antigravity/") {
		return userAgent
	}
	// Replaces "antigravity/X.Y.Z" with "antigravity/<version>"
	// Quick replacement via strings.Split or Regexp (here simple string split / index check)
	idx := strings.Index(userAgent, "antigravity/")
	if idx == -1 {
		return userAgent
	}

	spaceIdx := strings.Index(userAgent[idx:], " ")
	var originalSegment string
	if spaceIdx == -1 {
		originalSegment = userAgent[idx:]
	} else {
		originalSegment = userAgent[idx : idx+spaceIdx]
	}

	return strings.Replace(userAgent, originalSegment, "antigravity/"+version, 1)
}

func applyIdeVersionOverride(body []byte, headers http.Header) ([]byte, http.Header) {
	// 1. Copy headers and update User-Agent
	nextHeaders := make(http.Header)
	for k, vv := range headers {
		nextHeaders[k] = vv
	}

	ua := nextHeaders.Get("User-Agent")
	if ua != "" {
		nextHeaders.Set("User-Agent", rewriteUserAgent(ua, AntigravityIdeVersion))
	}

	// 2. Decode body and update metadata.ideVersion if applicable
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err == nil {
		if metaRaw, ok := payload["metadata"]; ok {
			metaBytes, _ := json.Marshal(metaRaw)
			var meta Metadata
			if err := json.Unmarshal(metaBytes, &meta); err == nil && shouldRewriteMetadata(&meta) {
				meta.IdeVersion = AntigravityIdeVersion
				payload["metadata"] = meta
				newBody, err := json.Marshal(payload)
				if err == nil {
					return newBody, nextHeaders
				}
			}
		}
	}

	return body, nextHeaders
}

// Intercept handles Google Antigravity chat completion requests and translates/executes them
func (h *AntigravityHandler) Intercept(w http.ResponseWriter, r *http.Request, bodyBuffer []byte, mappedModel string) {
	isStream := strings.Contains(r.URL.Path, ":streamGenerateContent")

	// Apply IDE version overrides
	finalBody, finalHeaders := applyIdeVersionOverride(bodyBuffer, r.Header)

	// In Node, we sent the raw Gemini request to /v1/chat/completions (which internally translated it).
	// In Go, since we want it lightweight, we can translate it here:
	// 1. Unmarshal into AntigravityRequest
	var agReq translator.AntigravityRequest
	if err := json.Unmarshal(finalBody, &agReq); err != nil {
		h.writeError(w, isStream, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
		return
	}

	// Override model if mappedModel was resolved (or fallback to default)
	if mappedModel != "" {
		agReq.Model = mappedModel
	}

	// 2. We can also handle Gemini -> OpenAI -> Gemini translation bypass if we want to run natively:
	// But wait, the IDE sent us a Gemini request, and we want to send a Gemini request to daily-cloudcode-pa!
	// Wait! The executor expects an AntigravityRequest already.
	// Oh, in the Node.js interceptor:
	// "intercept(req, res, bodyBuffer, mappedModel) { ... fetchRouter(body, '/v1/chat/completions', req.headers) }"
	// Wait, the Node interceptor forwarded the *Gemini* body to `/v1/chat/completions`!
	// How did `/v1/chat/completions` handle that?
	// Look at Node `open-sse/services/provider.js`:
	// `detectFormat(body)`: if `body.request?.contents` and `userAgent === "antigravity"`, format is "antigravity".
	// And if it needs translation, it translates it. But wait, if the source is Antigravity (Gemini-format) and the target provider is also Antigravity (Gemini-format),
	// does it translate?
	// In Node, `open-sse/translator/index.js` checks if translation is needed:
	// If `sourceFormat === targetFormat`, no translation occurs (or simple copy).
	// So in Go: if the incoming request is already in Gemini format, and we are forwarding to daily-cloudcode-pa (which is Gemini format),
	// we do not need to convert to OpenAI! We can forward the Gemini payload *directly* to Google.
	// This makes it incredibly fast!

	// Setup internal request ID and UUIDs
	agReq.UserAgent = "antigravity"
	agReq.RequestType = "agent"
	if agReq.RequestID == "" {
		agReq.RequestID = fmt.Sprintf("agent-%s", antigravity.GenerateProjectID())
	}
	if agReq.Project == "" {
		agReq.Project = antigravity.GenerateProjectID()
	}

	// We copy headers
	agReqDetail := agReq.Request
	sessionID := finalHeaders.Get("X-Machine-Session-Id")
	if sessionID == "" {
		sessionID = agReqDetail.SessionID
	}

	// Execute directly via executor (avoiding loopback HTTP overhead to /v1/chat/completions)
	ctx := context.WithValue(r.Context(), "headers", finalHeaders) // pass through if needed
	resp, err := h.Executor.Execute(ctx, &agReq, isStream)
	if err != nil {
		h.writeError(w, isStream, http.StatusBadGateway, fmt.Sprintf("Upstream execution error: %v", err))
		return
	}
	defer resp.Body.Close()

	// Replicate response headers and status
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Stream/copy response body directly back to client
	_, _ = io.Copy(w, resp.Body)
}

func (h *AntigravityHandler) writeError(w http.ResponseWriter, isStream bool, status int, msg string) {
	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		errPayload := map[string]interface{}{
			"error": map[string]string{
				"message": msg,
				"type":    "mitm_error",
			},
		}
		bytes, _ := json.Marshal(errPayload)
		_, _ = w.Write([]byte(fmt.Sprintf("data: %s\r\n\r\n", string(bytes))))
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		errPayload := map[string]interface{}{
			"error": map[string]string{
				"message": msg,
				"type":    "mitm_error",
			},
		}
		bytes, _ := json.Marshal(errPayload)
		_, _ = w.Write(bytes)
	}
}
