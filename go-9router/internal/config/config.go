package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	APIKey         string // Optional API Key to secure 9router endpoints
	Username       string // Username for Basic Auth
	Password       string // Password for Basic Auth
	ProxyURL       string // Optional proxy (SOCKS5/HTTP) for google requests
	ProxyPoolURLs  string // Comma-separated list of proxy URLs
	VercelRelayURL string // Optional URL for Vercel Serverless Relay
	StrictProxy    bool   // Fail if proxy fails
	DefaultModel   string
}

func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "20128" // Default port matching 9router
	}

	strictProxy, _ := strconv.ParseBool(os.Getenv("STRICT_PROXY"))

	defaultModel := os.Getenv("DEFAULT_MODEL")
	if defaultModel == "" {
		defaultModel = "claude-3-5-sonnet" // Or suitable default
	}

	return &Config{
		Port:           port,
		APIKey:         os.Getenv("API_KEY"),
		Username:       os.Getenv("AUTH_USER"),
		Password:       os.Getenv("AUTH_PASSWORD"),
		ProxyURL:       os.Getenv("PROXY_URL"),
		ProxyPoolURLs:  os.Getenv("PROXY_POOL_URLS"),
		VercelRelayURL: os.Getenv("VERCEL_RELAY_URL"),
		StrictProxy:    strictProxy,
		DefaultModel:   defaultModel,
	}
}
