package login

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// TokenStore defines the interface for token storage.
type TokenStore interface {
	// Save saves the token for an account.
	Save(accountID string, token *TokenInfo) error
	// Load loads the token for an account.
	Load(accountID string) (*TokenInfo, error)
	// Delete removes the token for an account.
	Delete(accountID string) error
	// List lists all stored account IDs.
	List() ([]string, error)
}

// TokenInfo contains stored token information.
type TokenInfo struct {
	Token     string `json:"token"`
	BaseURL   string `json:"base_url,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	SavedAt   string `json:"saved_at,omitempty"`
}

// FileTokenStore implements TokenStore using file system.
type FileTokenStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFileTokenStore creates a new file-based token store.
// If baseDir is empty, it defaults to ~/.weixin/
func NewFileTokenStore(baseDir string) (*FileTokenStore, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		baseDir = filepath.Join(home, ".weixin")
	}

	// Ensure directory exists
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}

	return &FileTokenStore{baseDir: baseDir}, nil
}

func (s *FileTokenStore) tokenPath(accountID string) string {
	return filepath.Join(s.baseDir, fmt.Sprintf("%s.json", accountID))
}

// Save saves the token for an account.
func (s *FileTokenStore) Save(accountID string, token *TokenInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	path := s.tokenPath(accountID)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// Load loads the token for an account.
func (s *FileTokenStore) Load(accountID string) (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.tokenPath(accountID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No token stored
		}
		return nil, fmt.Errorf("read file: %w", err)
	}

	var token TokenInfo
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}

	return &token, nil
}

// Delete removes the token for an account.
func (s *FileTokenStore) Delete(accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.tokenPath(accountID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

// List lists all stored account IDs.
func (s *FileTokenStore) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var accounts []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			name := entry.Name()
			accounts = append(accounts, name[:len(name)-5]) // Remove .json
		}
	}
	return accounts, nil
}

// MemoryTokenStore implements TokenStore in memory.
type MemoryTokenStore struct {
	tokens map[string]*TokenInfo
	mu     sync.RWMutex
}

// NewMemoryTokenStore creates a new in-memory token store.
func NewMemoryTokenStore() *MemoryTokenStore {
	return &MemoryTokenStore{
		tokens: make(map[string]*TokenInfo),
	}
}

// Save saves the token for an account.
func (s *MemoryTokenStore) Save(accountID string, token *TokenInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[accountID] = token
	return nil
}

// Load loads the token for an account.
func (s *MemoryTokenStore) Load(accountID string) (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokens[accountID], nil
}

// Delete removes the token for an account.
func (s *MemoryTokenStore) Delete(accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, accountID)
	return nil
}

// List lists all stored account IDs.
func (s *MemoryTokenStore) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]string, 0, len(s.tokens))
	for id := range s.tokens {
		accounts = append(accounts, id)
	}
	return accounts, nil
}