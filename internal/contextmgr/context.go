package contextmgr

import (
	"github.com/the-yex/wechat-ilink-sdk/internal/t"
)

// ContextTokenManager manages context tokens for message replies.
// Context tokens are required to send replies that are associated
// with a conversation context.
// Single-account design: key is userID only.
type ContextTokenManager struct {
	store *t.Map[string, string] // key: userID -> contextToken
}

// NewContextTokenManager creates a new context token manager.
func NewContextTokenManager() *ContextTokenManager {
	return &ContextTokenManager{
		store: t.New[string, string](),
	}
}

// Set stores a context token for a user.
func (m *ContextTokenManager) Set(userID, token string) {
	m.store.Store(userID, token)
}

// Get retrieves a context token for a user.
// Returns empty string if not found.
func (m *ContextTokenManager) Get(userID string) string {
	token, _ := m.store.Load(userID)
	return token
}

// Delete removes a context token for a user.
func (m *ContextTokenManager) Delete(userID string) {
	m.store.Delete(userID)
}

// Clear removes all context tokens.
func (m *ContextTokenManager) Clear() {
	m.store = t.New[string, string]()
}