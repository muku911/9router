package mitm

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"go-9router/internal/config"
	"go-9router/internal/db"
)

type Server struct {
	mu           sync.Mutex
	certManager  *CertManager
	agHandler    *AntigravityHandler
	httpServer   *http.Server
	dnsIPs       map[string]string // hostname -> real IP cache
	dnsCacheTTL  time.Duration
	running      bool
	proxyURL     string
	strictProxy  bool
	DBStore      *db.Store
}

func NewServer(cm *CertManager, agh *AntigravityHandler, proxyURL string, strictProxy bool, dbStore *db.Store) *Server {
	return &Server{
		certManager: cm,
		agHandler:   agh,
		dnsIPs:      make(map[string]string),
		dnsCacheTTL: 5 * time.Minute,
		proxyURL:    proxyURL,
		strictProxy: strictProxy,
		DBStore:     dbStore,
	}
}

func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	// 1. Setup TLS config with dynamic certificate resolver (SNI)
	tlsConfig := &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			serverName := hello.ServerName
			if serverName == "" {
				// Fallback to default
				serverName = "cloudcode-pa.googleapis.com"
			}
			return s.certManager.GenerateLeafCert(serverName)
		},
		MinVersion: tls.VersionTLS12,
	}

	// 2. Setup server mux and routing
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleMitmRequest)

	s.httpServer = &http.Server{
		Addr:      ":443", // MITM runs on standard HTTPS port
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	go func() {
		log.Println("MITM Server starting on :443...")
		// We use ListenAndServeTLS with empty cert files because TLSConfig.GetCertificate resolves them dynamically
		err := s.httpServer.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			log.Printf("MITM Server failed: %v\n", err)
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.httpServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		_ = s.httpServer.Close()
	}

	s.running = false
	log.Println("MITM Server stopped.")
	return nil
}

// Resolve DNS directly using Google DNS to bypass our poisoned local /etc/hosts file
func (s *Server) resolveRealIP(hostname string) (string, error) {
	s.mu.Lock()
	if ip, ok := s.dnsIPs[hostname]; ok {
		s.mu.Unlock()
		return ip, nil
	}
	s.mu.Unlock()

	// Resolve using 8.8.8.8
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}

	ips, err := r.LookupHost(context.Background(), hostname)
	if err != nil || len(ips) == 0 {
		return "", fmt.Errorf("failed resolving %s: %w", hostname, err)
	}

	realIP := ips[0]

	s.mu.Lock()
	s.dnsIPs[hostname] = realIP
	s.mu.Unlock()

	// Evict from cache after TTL
	go func() {
		time.Sleep(s.dnsCacheTTL)
		s.mu.Lock()
		delete(s.dnsIPs, hostname)
		s.mu.Unlock()
	}()

	return realIP, nil
}

func (s *Server) handleMitmRequest(w http.ResponseWriter, r *http.Request) {
	originalHost := r.Host
	if originalHost == "" {
		originalHost = "cloudcode-pa.googleapis.com"
	}
	cleanHost := strings.Split(originalHost, ":")[0]

	// Read body payload
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Check if this is a chat completion path
	isChatPath := strings.Contains(r.URL.Path, ":generateContent") || strings.Contains(r.URL.Path, ":streamGenerateContent")

	// If not chat, transparently pass through to upstream Google API
	if !isChatPath {
		s.passthrough(w, r, bodyBytes, cleanHost)
		return
	}

	// It's a chat path! We parse and intercept
	// Extract model name from URL or body
	model := extractModelFromRequest(r, bodyBytes)
	mappedModel := s.getMappedModel(model)

	if mappedModel == "" {
		// Passthrough if model isn't mapped
		s.passthrough(w, r, bodyBytes, cleanHost)
		return
	}

	// Intercept and handle natively
	s.agHandler.Intercept(w, r, bodyBytes, mappedModel)
}

func extractModelFromRequest(r *http.Request, body []byte) string {
	// 1. URL Path check (models/gemini-...)
	// URL path usually contains /v1internal/models/gemini-1.5-pro:generateContent
	regex := regexp.MustCompile(`/models/([^/:]+)`)
	matches := regex.FindStringSubmatch(r.URL.Path)
	if len(matches) > 1 {
		return matches[1]
	}

	// 2. Body model check
	var bodyJSON struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &bodyJSON); err == nil && bodyJSON.Model != "" {
		return bodyJSON.Model
	}

	return ""
}

// Synonyms map modeled after Node's config.js + dynamic mappings from database
func (s *Server) getMappedModel(model string) string {
	if model == "" {
		return ""
	}

	modelClean := strings.TrimPrefix(model, "models/")

	// 1. Try dynamic alias from database if available
	if s.DBStore != nil {
		payload := s.DBStore.GetPayload()
		if payload.MitmAlias != nil {
			if val, ok := payload.MitmAlias["antigravity"]; ok {
				if mapping, ok := val.(map[string]interface{}); ok {
					if targetModel, ok := mapping[modelClean]; ok {
						if targetStr, ok := targetModel.(string); ok && targetStr != "" {
							return targetStr
						}
					}
				}
			}
		}
	}

	// 2. Fallback to hardcoded synonyms map
	synonyms := map[string]string{
		"claude-3-5-sonnet":           "gemini-3.5-sonnet-agent",
		"claude-3.5-sonnet":           "gemini-3.5-sonnet-agent",
		"claude-3-opus":               "gemini-3-opus-agent",
		"claude-3-haiku":              "gemini-3-haiku-agent",
		"gemini-1.5-pro":              "gemini-1.5-pro",
		"gemini-1.5-flash":            "gemini-1.5-flash",
		"gemini-pro-experimental":     "gemini-pro-experimental",
		"gemini-2.0-flash-exp":        "gemini-2.0-flash-exp",
		"gemini-2.5-pro-exp":          "gemini-2.5-pro-exp",
		"gemini-3.5-sonnet-agent":     "gemini-3.5-sonnet-agent",
	}

	if val, ok := synonyms[modelClean]; ok {
		return val
	}

	// If no synonym match, just return the model Clean itself
	return modelClean
}

// Transparent HTTP/1.1 or HTTP/2 Reverse Proxy resolving real IP to bypass poisoned local hosts
func (s *Server) passthrough(w http.ResponseWriter, r *http.Request, bodyBytes []byte, hostname string) {
	// Map cloudcode-pa.googleapis.com to daily-cloudcode-pa.googleapis.com for chat endpoints if requested
	targetHost := hostname
	isChatEndpoint := strings.Contains(r.URL.Path, ":generateContent") || strings.Contains(r.URL.Path, ":streamGenerateContent")
	if isChatEndpoint && hostname == "cloudcode-pa.googleapis.com" {
		targetHost = "daily-cloudcode-pa.googleapis.com"
	}

	realIP, err := s.resolveRealIP(targetHost)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error": "DNS resolution failed: %v"}`, err)))
		return
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			ServerName:         targetHost,
			InsecureSkipVerify: true, // We are doing MITM, so we trust Google
		},
	}

	// Setup proxy if configured
	if s.proxyURL != "" {
		proxyURI, err := url.Parse(s.proxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURI)
		} else if s.strictProxy {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error": "Strict proxy configured but proxy URL parse failed"}`))
			return
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	targetURL := fmt.Sprintf("https://%s%s", realIP, r.URL.RequestURI())
	req, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Copy headers
	for k, vv := range r.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	// Overwrite Host header to target
	req.Host = targetHost
	req.Header.Set("Host", targetHost)

	// Inject internal request header to avoid recursion loops
	req.Header.Set(config.InternalRequestHeaderName, config.InternalRequestHeaderValue)

	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error": "Passthrough failed: %v"}`, err)))
		return
	}
	defer resp.Body.Close()

	// Forward response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Pipe body
	_, _ = io.Copy(w, resp.Body)
}
