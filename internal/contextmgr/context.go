package contextmgr

import (
	"github.com/the-yex/wechat-ilink-sdk/internal/t"
)

// ContextTokenManager manages context tokens for message replies.
// Context tokens are required to send replies that are associated
// with a conversation context.
type ContextTokenManager struct {
	store *t.Map[string, string] // key: accountID:userID -> contextToken
}

// NewContextTokenManager creates a new context token manager.
func NewContextTokenManager() *ContextTokenManager {
	return &ContextTokenManager{
		store: t.New[string, string](),
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
	m.store.Store(contextTokenKey(accountID, userID), token)
}

// Get retrieves a context token.
// Returns empty string if not found.
func (m *ContextTokenManager) Get(accountID, userID string) string {
	token, _ := m.store.Load(contextTokenKey(accountID, userID))
	return token
}

// Delete removes a context token.
func (m *ContextTokenManager) Delete(accountID, userID string) {
	m.store.Delete(contextTokenKey(accountID, userID))
}

// Clear removes all context tokens.
func (m *ContextTokenManager) Clear() {
	m.store = t.New[string, string]()
}