package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ExecuteResult holds the upstream response from executing a request.
type ExecuteResult struct {
	Response    *http.Response
	URL         string
	ContentType string
}

// Credentials holds auth info for a connection.
type Credentials struct {
	APIKey      string
	AccessToken string
}

// Execute sends a request to the upstream provider, handling format translation,
// URL building, and auth headers automatically.
func Execute(ctx context.Context, provider string, model string, body map[string]interface{}, stream bool, creds Credentials, httpClient *http.Client) (*ExecuteResult, error) {
	cfg := GetProviderConfig(provider)
	targetFormat := cfg.Format

	// Translate request from OpenAI format to target provider format
	translatedBody := TranslateRequest(targetFormat, model, body, stream)

	// Build URL
	url := buildProviderURL(provider, cfg, model, stream)

	// Build headers
	headers := buildProviderHeaders(provider, cfg, creds, stream)

	// Marshal body
	bodyBytes, err := json.Marshal(translatedBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Use provided client or default
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request to %s: %w", provider, err)
	}

	return &ExecuteResult{
		Response:    resp,
		URL:         url,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

// buildProviderURL constructs the endpoint URL for a provider.
func buildProviderURL(provider string, cfg ProviderConfig, model string, stream bool) string {
	// OpenAI-compatible custom providers
	if strings.HasPrefix(provider, "openai-compatible-") {
		return cfg.BaseURL // Base URL already includes /chat/completions
	}
	if strings.HasPrefix(provider, "anthropic-compatible-") {
		return cfg.BaseURL
	}

	switch provider {
	case "gemini":
		action := "generateContent"
		if stream {
			action = "streamGenerateContent?alt=sse"
		}
		return fmt.Sprintf("%s%s:%s", cfg.BaseURL, model, action)

	case "anthropic", "claude":
		return cfg.BaseURL

	default:
		return cfg.BaseURL
	}
}

// buildProviderHeaders constructs auth and content headers.
func buildProviderHeaders(provider string, cfg ProviderConfig, creds Credentials, stream bool) map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// Merge static headers from config
	for k, v := range cfg.Headers {
		headers[k] = v
	}

	// Auth
	token := creds.AccessToken
	if token == "" {
		token = creds.APIKey
	}

	switch cfg.AuthStyle {
	case "x-api-key":
		headers["x-api-key"] = token
	case "x-goog-api-key":
		headers["x-goog-api-key"] = token
	default: // "bearer"
		headers["Authorization"] = "Bearer " + token
	}

	if stream {
		headers["Accept"] = "text/event-stream"
	}

	return headers
}

// StreamSSE reads an SSE stream from resp and calls onChunk for each data line.
// It handles the `data: ...` prefix and ignores comments/empty lines.
func StreamSSE(resp *http.Response, onChunk func(data string) error) error {
	defer resp.Body.Close()
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)

	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			// Process complete lines
			for {
				idx := bytes.IndexByte(buf, '\n')
				if idx < 0 {
					break
				}
				line := string(buf[:idx])
				buf = buf[idx+1:]
				line = strings.TrimRight(line, "\r")

				if line == "" || strings.HasPrefix(line, ":") {
					continue // Skip empty lines and comments
				}

				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					if data == "[DONE]" {
						return nil
					}
					if chunkErr := onChunk(data); chunkErr != nil {
						return chunkErr
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				// Process remaining buffer
				if len(buf) > 0 {
					line := strings.TrimRight(string(buf), "\r\n")
					if strings.HasPrefix(line, "data: ") {
						data := strings.TrimPrefix(line, "data: ")
						if data != "[DONE]" {
							_ = onChunk(data)
						}
					}
				}
				return nil
			}
			return err
		}
	}
}
