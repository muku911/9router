package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-9router/internal/config"
	"go-9router/internal/db"
	"go-9router/internal/translator"
)

type Executor struct {
	mu             sync.RWMutex
	TokenStore     *TokenStore
	HTTPClient     *http.Client
	VercelRelayURL string
	DBStore        *db.Store
	ConnectionID   string // The active connection ID being run
}

func NewExecutor(store *TokenStore, client *http.Client, vercelRelayURL string, dbStore *db.Store, connID string) *Executor {
	return &Executor{
		TokenStore:     store,
		HTTPClient:     client,
		VercelRelayURL: vercelRelayURL,
		DBStore:        dbStore,
		ConnectionID:   connID,
	}
}

func (e *Executor) SwapDependencies(store *TokenStore, client *http.Client, vercelRelayURL string, connID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.TokenStore = store
	e.HTTPClient = client
	e.VercelRelayURL = vercelRelayURL
	e.ConnectionID = connID
}

// Generate Google-style project name: Swift-wave-12345
func GenerateProjectID() string {
	adjs := []string{"useful", "bright", "swift", "calm", "bold"}
	nouns := []string{"fuze", "wave", "spark", "flow", "core"}
	now := time.Now().UnixNano()
	adj := adjs[now%int64(len(adjs))]
	noun := nouns[(now/2)%int64(len(nouns))]
	return fmt.Sprintf("%s-%s-%x", adj, noun, now%0xfffff)
}

func (e *Executor) BuildURL(stream bool) string {
	action := "generateContent"
	if stream {
		action = "streamGenerateContent?alt=sse"
	}
	return fmt.Sprintf("%s/v1internal:%s", config.AntigravityBase, action)
}

func (e *Executor) BuildHeaders(token string, stream bool, sessionID string) http.Header {
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("Authorization", "Bearer "+token)
	headers.Set("User-Agent", config.GetUserAgent())
	headers.Set(config.InternalRequestHeaderName, config.InternalRequestHeaderValue)

	if sessionID != "" {
		headers.Set("X-Machine-Session-Id", sessionID)
	}

	if stream {
		headers.Set("Accept", "text/event-stream")
	} else {
		headers.Set("Accept", "application/json")
	}

	return headers
}

type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		Details []struct {
			Type       string            `json:"@type"`
			Reason     string            `json:"reason,omitempty"`
			Metadata   map[string]string `json:"metadata,omitempty"`
			RetryDelay string            `json:"retryDelay,omitempty"`
		} `json:"details"`
	} `json:"error"`
}

// Parse rate limit reset or cooldown details from error response body or headers
func (e *Executor) ParseRetryTime(resp *http.Response, body []byte) time.Duration {
	// 1. Try body Details
	var errRes ErrorResponse
	if err := json.Unmarshal(body, &errRes); err == nil && errRes.Error.Code != 0 {
		for _, d := range errRes.Error.Details {
			// Extract retry delay from RetryInfo detail
			if d.Type == "type.googleapis.com/google.rpc.RetryInfo" && d.RetryDelay != "" {
				s := strings.TrimSuffix(d.RetryDelay, "s")
				if val, err := strconv.ParseFloat(s, 64); err == nil && val > 0 {
					return time.Duration(val * float64(time.Second))
				}
			}
			// Extract from quotaResetTimeStamp
			if d.Type == "type.googleapis.com/google.rpc.ErrorInfo" && d.Metadata != nil {
				if tsStr, ok := d.Metadata["quotaResetTimeStamp"]; ok {
					if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
						diff := time.Until(t)
						if diff > 0 {
							return diff
						}
					}
				}
			}
		}

		// Parse from message text (e.g. "Your quota will reset after 2h7m23s")
		msg := errRes.Error.Message
		if diff := parseRetryFromMessage(msg); diff > 0 {
			return diff
		}
	}

	// 2. Try headers
	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if val, err := strconv.Atoi(retryAfter); err == nil && val > 0 {
				return time.Duration(val) * time.Second
			}
		}
		if resetAfter := resp.Header.Get("X-RateLimit-Reset-After"); resetAfter != "" {
			if val, err := strconv.Atoi(resetAfter); err == nil && val > 0 {
				return time.Duration(val) * time.Second
			}
		}
	}

	return 0
}

var retryMsgRegex = regexp.MustCompile(`(?i)(?:reset after|resets in) (?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?`)

func parseRetryFromMessage(msg string) time.Duration {
	match := retryMsgRegex.FindStringSubmatch(msg)
	if match == nil {
		return 0
	}

	var duration time.Duration
	if match[1] != "" {
		h, _ := strconv.Atoi(match[1])
		duration += time.Duration(h) * time.Hour
	}
	if match[2] != "" {
		m, _ := strconv.Atoi(match[2])
		duration += time.Duration(m) * time.Minute
	}
	if match[3] != "" {
		s, _ := strconv.Atoi(match[3])
		duration += time.Duration(s) * time.Second
	}
	return duration
}

// GetQuotaFamily resolves the quota family for locking (Gemini models vs Claude/others)
func GetQuotaFamily(model string) string {
	cleaned := strings.ToLower(model)
	if strings.Contains(cleaned, "gemini") {
		return "ag-gemini"
	}
	return "ag-non-gemini"
}

func (e *Executor) Execute(ctx context.Context, agReq *translator.AntigravityRequest, stream bool) (*http.Response, error) {
	e.mu.RLock()
	store := e.TokenStore
	client := e.HTTPClient
	vercelRelayURL := e.VercelRelayURL
	dbStore := e.DBStore
	connID := e.ConnectionID
	e.mu.RUnlock()

	model := agReq.Model
	family := GetQuotaFamily(model)

	// Check if this connection is currently locked for the requested model/family
	if dbStore != nil && connID != "" {
		if err := dbStore.CheckLock(connID, model, family); err != nil {
			log.Printf("[Quota Lock Alert] Connection %s is locked for model %s (family %s). Skipping execution.\n", connID, model, family)
			return nil, fmt.Errorf("connection locked: %w", err)
		}
	}

	log.Printf("[Request Dispatch] Executing request for model %s via connection %s\n", model, connID)

	token, err := store.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain token: %w", err)
	}

	reqBody, err := json.Marshal(agReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Calculate endpoint details (for Vercel Relay headers if active)
	action := "generateContent"
	if stream {
		action = "streamGenerateContent?alt=sse"
	}

	urlStr := e.BuildURL(stream)
	headers := e.BuildHeaders(token, stream, agReq.Request.SessionID)

	// Vercel Serverless Relay support:
	// If VercelRelayURL is configured, route request to the relay URL
	// and embed target details via relay headers
	if vercelRelayURL != "" {
		urlStr = vercelRelayURL
		headers.Set("x-relay-target", config.AntigravityBase)
		headers.Set("x-relay-path", fmt.Sprintf("/v1internal:%s", action))
	}

	var lastErr error
	var resp *http.Response

	maxAutoRetries := 3
	for attempt := 0; attempt <= maxAutoRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create http request: %w", err)
		}

		// Set calculated headers
		req.Header = headers

		resp, err = client.Do(req)
		if err != nil {
			lastErr = err
			// Network error, try again
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(math.Pow(2, float64(attempt))) * time.Second):
				continue
			}
		}

		// If unauthorized, token might have expired, clear token and retry immediately once
		if resp.StatusCode == http.StatusUnauthorized && attempt < maxAutoRetries {
			resp.Body.Close()
			store.mu.Lock()
			store.accessToken = "" // force refresh
			store.mu.Unlock()

			token, err = store.GetAccessToken()
			if err != nil {
				return nil, fmt.Errorf("token refresh during 401 fail: %w", err)
			}
			// Update headers authorization token
			headers.Set("Authorization", "Bearer "+token)
			continue
		}

		// Rate limited (429) or Service Unavailable (503)
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			// Read body to parse retry duration
			bodyCopy := bytes.Buffer{}
			tee := io.TeeReader(resp.Body, &bodyCopy)
			bodyBytes, _ := io.ReadAll(tee)
			resp.Body.Close()

			retryDelay := e.ParseRetryTime(resp, bodyBytes)

			// If rate limited and retry delay is large, or we exhausted retries, apply Lock Mode
			if retryDelay > 10*time.Second || attempt == maxAutoRetries {
				lockDuration := retryDelay
				if lockDuration <= 0 {
					lockDuration = 30 * time.Minute // default fallback lock
				}
				if dbStore != nil && connID != "" {
					log.Printf("Locking connection %s for model %s (family %s) for %v\n", connID, model, family, lockDuration)
					dbStore.ApplyLock(connID, model, family, lockDuration)
				}
			}

			// limit delay to max 10s for auto retries in Go
			if retryDelay > 0 && retryDelay <= 10*time.Second && attempt < maxAutoRetries {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(retryDelay):
					continue
				}
			}

			// If no explicit retry time or too long, do exponential backoff up to 10s
			if attempt < maxAutoRetries {
				backoff := time.Duration(math.Min(float64(time.Duration(math.Pow(2, float64(attempt+1)))*time.Second), float64(10*time.Second)))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
					continue
				}
			}

			return nil, fmt.Errorf("request rate limited or unavailable with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("antigravity API returned status %d: %s", resp.StatusCode, string(body))
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all retries failed: %w", lastErr)
	}
	return nil, fmt.Errorf("all retries failed")
}
