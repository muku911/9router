package antigravity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go-9router/internal/config"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type TokenStore struct {
	mu           sync.RWMutex
	accessToken  string
	refreshToken string
	expiry       time.Time
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

func NewTokenStore(refreshToken, clientID, clientSecret string, client *http.Client) *TokenStore {
	return &TokenStore{
		refreshToken: refreshToken,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   client,
	}
}

func (s *TokenStore) GetAccessToken() (string, error) {
	s.mu.RLock()
	// If token is still valid (with 2-minute buffer)
	if s.accessToken != "" && time.Now().Add(2*time.Minute).Before(s.expiry) {
		token := s.accessToken
		s.mu.RUnlock()
		return token, nil
	}
	s.mu.RUnlock()

	// Needs refresh
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check if another goroutine refreshed it
	if s.accessToken != "" && time.Now().Add(2*time.Minute).Before(s.expiry) {
		return s.accessToken, nil
	}

	if s.refreshToken == "" {
		return "", fmt.Errorf("no refresh token provided")
	}

	// Make refresh request
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", s.refreshToken)
	data.Set("client_id", s.clientID)
	data.Set("client_secret", s.clientSecret)

	req, err := http.NewRequest("POST", config.GoogleTokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create refresh token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenRes TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	s.accessToken = tokenRes.AccessToken
	if tokenRes.RefreshToken != "" {
		s.refreshToken = tokenRes.RefreshToken
	}
	s.expiry = time.Now().Add(time.Duration(tokenRes.ExpiresIn) * time.Second)

	return s.accessToken, nil
}

func (s *TokenStore) UpdateCredentials(accessToken, refreshToken string, expiresIn int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessToken = accessToken
	if refreshToken != "" {
		s.refreshToken = refreshToken
	}
	s.expiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
}

func (s *TokenStore) SwapCredentials(refreshToken, clientID, clientSecret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshToken = refreshToken
	s.clientID = clientID
	s.clientSecret = clientSecret
	s.accessToken = "" // invalidate cache to force refresh on next check
}
