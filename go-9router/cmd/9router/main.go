package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go-9router/internal/api"
	"go-9router/internal/config"
	"go-9router/internal/db"
	"go-9router/internal/mitm"
	"go-9router/internal/provider/antigravity"
	"go-9router/internal/utils"
)

func main() {
	// CLI Flags setup
	portFlag := flag.String("port", "", "Port to run on (overrides PORT env var)")
	apiKeyFlag := flag.String("api-key", "", "API Key to secure endpoints (overrides API_KEY env var)")
	userFlag := flag.String("user", "", "Username for Basic Auth (overrides AUTH_USER env var)")
	passFlag := flag.String("password", "", "Password for Basic Auth (overrides AUTH_PASSWORD env var)")
	refreshTokenFlag := flag.String("refresh-token", "", "Google OAuth Refresh Token (overrides REFRESH_TOKEN env var)")
	clientIdFlag := flag.String("client-id", "", "Google Client ID (overrides CLIENT_ID env var)")
	clientSecretFlag := flag.String("client-secret", "", "Google Client Secret (overrides CLIENT_SECRET env var)")
	proxyFlag := flag.String("proxy", "", "Proxy URL (overrides PROXY_URL env var)")
	proxyPoolFlag := flag.String("proxy-pool", "", "Comma-separated list of proxy URLs (overrides PROXY_POOL_URLS env var)")
	vercelRelayFlag := flag.String("vercel-relay", "", "Vercel Serverless Relay URL (overrides VERCEL_RELAY_URL env var)")
	mitmFlag := flag.Bool("mitm", false, "Start MITM server directly on startup (requires root/admin privileges for port 443)")

	flag.Parse()

	cfg := config.LoadConfig()

	// Initialize Console Log Buffer
	logBuffer := utils.NewLogBuffer(500)
	multiWriter := io.MultiWriter(os.Stdout, logBuffer)
	log.SetOutput(multiWriter)

	// 1. Initialize JSON DB Store
	store, err := db.NewStore("")
	if err != nil {
		log.Fatalf("Failed to initialize JSON DB: %v\n", err)
	}

	// Dynamic dependency components
	var tokenStore *antigravity.TokenStore
	var executor *antigravity.Executor
	var mitmServer *mitm.Server

	// Reusable direct http client for token refreshing (OAuth bypasses proxies/relays)
	directHttpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
		Timeout: 30 * time.Second,
	}

	// Function to parse config from Env -> CLI -> DB and rebuild/reload the dependencies
	reloadConfigAndDependencies := func() {
		// Read settings database data
		payload := store.GetPayload()

		// A. Evaluate Overrides from CLI, Env & DB Settings
		finalPort := cfg.Port
		if *portFlag != "" {
			finalPort = *portFlag
		}

		finalApiKey := cfg.APIKey
		if *apiKeyFlag != "" {
			finalApiKey = *apiKeyFlag
		}

		finalUsername := cfg.Username
		if *userFlag != "" {
			finalUsername = *userFlag
		}

		finalPassword := cfg.Password
		if *passFlag != "" {
			finalPassword = *passFlag
		}

		// Check database settings
		var dbProxy string
		if payload.Settings != nil {
			if proxyVal, ok := payload.Settings["outboundProxyUrl"]; ok {
				dbProxy = strings.TrimSpace(proxyVal.(string))
			}
		}

		// Resolve proxy pool
		var proxyList []string
		if *proxyPoolFlag != "" {
			proxyList = strings.Split(*proxyPoolFlag, ",")
		} else if cfg.ProxyPoolURLs != "" {
			proxyList = strings.Split(cfg.ProxyPoolURLs, ",")
		} else {
			// Read active proxies from DB
			for _, pool := range payload.ProxyPools {
				if pool.IsActive {
					if urlVal, ok := pool.Data["url"]; ok {
						proxyList = append(proxyList, strings.TrimSpace(urlVal.(string)))
					}
				}
			}
		}

		// Fallback to single proxy if pool is empty
		if len(proxyList) == 0 {
			if *proxyFlag != "" {
				proxyList = []string{*proxyFlag}
			} else if dbProxy != "" {
				proxyList = []string{dbProxy}
			} else if cfg.ProxyURL != "" {
				proxyList = []string{cfg.ProxyURL}
			}
		}

		// Clean slice items
		for i, p := range proxyList {
			proxyList[i] = strings.TrimSpace(p)
		}

		// B. Evaluate Credentials (refresh token, etc.)
		finalRefreshToken := os.Getenv("REFRESH_TOKEN")
		if *refreshTokenFlag != "" {
			finalRefreshToken = *refreshTokenFlag
		}

		finalClientID := os.Getenv("CLIENT_ID")
		if *clientIdFlag != "" {
			finalClientID = *clientIdFlag
		}

		finalClientSecret := os.Getenv("CLIENT_SECRET")
		if *clientSecretFlag != "" {
			finalClientSecret = *clientSecretFlag
		}

		// Override credentials with active DB Provider Connections if they exist
		activeConnID := ""
		for _, conn := range payload.ProviderConnections {
			if conn.Provider == "antigravity" && conn.IsActive {
				activeConnID = conn.ID
				// Parse custom fields from connection details
				if token, ok := conn.Data["refreshToken"]; ok && token.(string) != "" {
					finalRefreshToken = token.(string)
				} else if conn.RefreshToken != "" {
					finalRefreshToken = conn.RefreshToken
				}

				if cid, ok := conn.Data["clientId"]; ok && cid.(string) != "" {
					finalClientID = cid.(string)
				} else if conn.ClientID != "" {
					finalClientID = conn.ClientID
				}

				if csec, ok := conn.Data["clientSecret"]; ok && csec.(string) != "" {
					finalClientSecret = csec.(string)
				} else if conn.ClientSecret != "" {
					finalClientSecret = conn.ClientSecret
				}
				break
			}
		}

		// Resolve Vercel Relay
		finalVercelRelay := cfg.VercelRelayURL
		if *vercelRelayFlag != "" {
			finalVercelRelay = *vercelRelayFlag
		}

		// Apply overrides back to runtime config
		cfg.Port = finalPort
		cfg.APIKey = finalApiKey
		cfg.Username = finalUsername
		cfg.Password = finalPassword

		// C. Build Proxy Transport
		var transport http.RoundTripper
		if len(proxyList) > 0 {
			poolTransport, err := utils.NewProxyPoolRoundTripper(proxyList, cfg.StrictProxy)
			if err != nil {
				log.Printf("ERROR: Failed to parse proxy pool URLs: %v\n", err)
			} else {
				transport = poolTransport
				log.Printf("Reload: Proxy Pool configured with %d proxies.\n", len(proxyList))
			}
		}

		if transport == nil {
			transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			}
		}

		httpClient := &http.Client{
			Transport: transport,
			Timeout:   120 * time.Second,
		}

		// D. Swapping dependencies atomically inside existing structures
		if tokenStore == nil {
			tokenStore = antigravity.NewTokenStore(finalRefreshToken, finalClientID, finalClientSecret, directHttpClient)
		} else {
			tokenStore.SwapCredentials(finalRefreshToken, finalClientID, finalClientSecret)
		}

		if executor == nil {
			executor = antigravity.NewExecutor(tokenStore, httpClient, finalVercelRelay, store, activeConnID)
		} else {
			executor.SwapDependencies(tokenStore, httpClient, finalVercelRelay, activeConnID)
		}

		log.Println("Configurations and dependencies reloaded successfully.")
	}

	// Run initial loading
	reloadConfigAndDependencies()

	// Setup MITM Server
	certManager, err := mitm.NewCertManager("")
	if err != nil {
		log.Fatalf("Failed to initialize CertManager: %v", err)
	}

	agMitmHandler := mitm.NewAntigravityHandler(executor)
	mitmServer = mitm.NewServer(certManager, agMitmHandler, cfg.ProxyURL, cfg.StrictProxy, store)

	// Start MITM Server on startup if flag set
	if *mitmFlag {
		log.Println("Starting MITM Server on startup (port 443)...")
		if err := mitmServer.Start(); err != nil {
			log.Printf("Failed to start MITM Server on port 443: %v (Ensure you are running with sudo/admin privileges)\n", err)
		} else {
			// Auto poison DNS on startup
			if err := mitm.AddDNSEntries(""); err != nil {
				log.Printf("Failed to update DNS entries: %v (Ensure you run with sudo/admin)\n", err)
			}
		}
	}

	// Setup HTTP Server
	mux := http.NewServeMux()
	chatHandler := api.NewChatHandler(executor)
	mitmHandler := api.NewMitmHandler(mitmServer, store)
	settingsHandler := api.NewSettingsHandler(store, reloadConfigAndDependencies)
	logHandler := api.NewLogHandler(logBuffer)
	handler := api.RegisterRoutes(mux, cfg, chatHandler, mitmHandler, settingsHandler, logHandler, store)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 125 * time.Second, // Keep high to allow long streams
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down go-9router server gracefully...")

		// Stop MITM if running
		if mitmServer.IsRunning() {
			_ = mitmServer.Stop()
		}
		mitm.RemoveAllDNSEntriesSync() // Clean up /etc/hosts entries sync

		_ = server.Close()
	}()

	log.Printf("go-9router server starting on port %s (bound to 0.0.0.0 for global network access)...\n", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server startup failed: %v", err)
	}
}
