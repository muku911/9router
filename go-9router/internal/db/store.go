package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// GetDataDir resolves platform-specific standard config directory for 9router (shared)
func GetDataDir() string {
	if configured := os.Getenv("DATA_DIR"); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "9router")
	}
	return filepath.Join(home, ".9router")
}

type ProviderConnection struct {
	ID           string                 `json:"id"`
	Provider     string                 `json:"provider"`
	AuthType     string                 `json:"authType"`
	Name         string                 `json:"name"`
	Email        string                 `json:"email"`
	Priority     int                    `json:"priority"`
	IsActive     bool                   `json:"isActive"`
	RefreshToken string                 `json:"refreshToken,omitempty"`
	AccessToken  string                 `json:"accessToken,omitempty"`
	ClientID     string                 `json:"clientId,omitempty"`
	ClientSecret string                 `json:"clientSecret,omitempty"`
	Locks        map[string]string      `json:"locks,omitempty"` // locks map holding expiration ISO strings (e.g. modelLock_<model>, modelGroupLock_<family>)
	Data         map[string]interface{} `json:"data,omitempty"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
}

type ProxyPool struct {
	ID         string                 `json:"id"`
	IsActive   bool                   `json:"isActive"`
	TestStatus string                 `json:"testStatus"`
	Data       map[string]interface{} `json:"data,omitempty"`
	CreatedAt  string                 `json:"createdAt"`
	UpdatedAt  string                 `json:"updatedAt"`
}

type SettingsData struct {
	OutboundProxyUrl string `json:"outboundProxyUrl,omitempty"`
	RequireLogin     bool   `json:"requireLogin,omitempty"`
}

type DatabasePayload struct {
	Settings            map[string]interface{}   `json:"settings"`
	ProviderConnections []ProviderConnection     `json:"providerConnections"`
	ProviderNodes       []map[string]interface{} `json:"providerNodes"`
	ProxyPools          []ProxyPool              `json:"proxyPools"`
	ApiKeys             []map[string]interface{} `json:"apiKeys"`
	Combos              []map[string]interface{} `json:"combos"`
	ModelAliases        map[string]interface{}   `json:"modelAliases"`
	CustomModels        []map[string]interface{} `json:"customModels"`
	MitmAlias           map[string]interface{}   `json:"mitmAlias"`
	Pricing             map[string]interface{}   `json:"pricing"`
}

type Store struct {
	mu       sync.RWMutex
	filePath string
	payload  *DatabasePayload
}

func NewStore(filePath string) (*Store, error) {
	if filePath == "" {
		filePath = filepath.Join(GetDataDir(), "db", "data.json")
	}

	dbDir := filepath.Dir(filePath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}

	// Trigger one-time SQLite migration if JSON file is missing
	if err := CheckAndMigrate(filePath); err != nil {
		log.Printf("SQLite database migration warning: %v\n", err)
	}

	s := &Store{
		filePath: filePath,
		payload: &DatabasePayload{
			Settings:            make(map[string]interface{}),
			ProviderConnections: []ProviderConnection{},
			ProxyPools:          []ProxyPool{},
			ModelAliases:        make(map[string]interface{}),
			MitmAlias:           make(map[string]interface{}),
			Pricing:             make(map[string]interface{}),
		},
	}

	if err := s.Load(); err != nil {
		// If file doesn't exist, we just write the default payload structure to disk
		if os.IsNotExist(err) {
			_ = s.Save()
		} else {
			return nil, err
		}
	}

	return s, nil
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var payload DatabasePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal JSON DB: %w", err)
	}

	// Make sure maps aren't nil
	if payload.Settings == nil {
		payload.Settings = make(map[string]interface{})
	}
	if payload.ModelAliases == nil {
		payload.ModelAliases = make(map[string]interface{})
	}
	if payload.MitmAlias == nil {
		payload.MitmAlias = make(map[string]interface{})
	}
	if payload.Pricing == nil {
		payload.Pricing = make(map[string]interface{})
	}

	s.payload = &payload
	return nil
}

func (s *Store) Save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.payload, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}

	// Atomic write using a temp file
	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Rename(tmpFile, s.filePath)
}

func (s *Store) Lock() {
	s.mu.Lock()
}

func (s *Store) Unlock() {
	s.mu.Unlock()
}

func (s *Store) RLock() {
	s.mu.RLock()
}

func (s *Store) RUnlock() {
	s.mu.RUnlock()
}

func (s *Store) GetPayload() *DatabasePayload {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.payload
}

// GetPayloadUnsafe returns the payload without acquiring any lock.
// The caller MUST hold s.Lock() before calling this.
func (s *Store) GetPayloadUnsafe() *DatabasePayload {
	return s.payload
}

// PatchSettings merges the given key-value pairs into the Settings map and persists.
func (s *Store) PatchSettings(patch map[string]interface{}) error {
	s.mu.Lock()
	for k, v := range patch {
		s.payload.Settings[k] = v
	}
	s.mu.Unlock()
	return s.Save()
}

func (s *Store) Import(data []byte) error {
	var payload DatabasePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("invalid backup format: %w", err)
	}

	s.mu.Lock()
	s.payload = &payload
	s.mu.Unlock()

	return s.Save()
}
