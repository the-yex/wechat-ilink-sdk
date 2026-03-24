package contextmgr

import (
	"sync"
)

// ContextTokenManager manages context tokens for message replies.
// Context tokens are required to send replies that are associated
// with a conversation context.
type ContextTokenManager struct {
	mu    sync.RWMutex
	store map[string]string // key: accountID:userID -> contextToken
}

// NewContextTokenManager creates a new context token manager.
func NewContextTokenManager() *ContextTokenManager {
	return &ContextTokenManager{
		store: make(map[string]string),
	}
}

// contextTokenKey generates the storage key for a context token.
func contextTokenKey(accountID, userID string) string {
	if accountID == "" {
		return userID
	}
	return accountID + ":" + userID
}

// Set stores a context token.
func (m *ContextTokenManager) Set(accountID, userID, token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[contextTokenKey(accountID, userID)] = token
}

// Get retrieves a context token.
func (m *ContextTokenManager) Get(accountID, userID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store[contextTokenKey(accountID, userID)]
}

// Delete removes a context token.
func (m *ContextTokenManager) Delete(accountID, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, contextTokenKey(accountID, userID))
}

// Clear removes all context tokens.
func (m *ContextTokenManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = make(map[string]string)
}
