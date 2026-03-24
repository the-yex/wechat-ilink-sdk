package contextmgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextTokenManager(t *testing.T) {
	manager := NewContextTokenManager()

	t.Run("set and get", func(t *testing.T) {
		manager.Set("account1", "user1", "token123")

		token := manager.Get("account1", "user1")
		assert.Equal(t, "token123", token)
	})

	t.Run("get non-existent", func(t *testing.T) {
		token := manager.Get("nonexistent", "user")
		assert.Equal(t, "", token)
	})

	t.Run("delete", func(t *testing.T) {
		manager.Set("account1", "user2", "token456")
		manager.Delete("account1", "user2")

		token := manager.Get("account1", "user2")
		assert.Equal(t, "", token)
	})

	t.Run("overwrite", func(t *testing.T) {
		manager.Set("account1", "user1", "newtoken")
		token := manager.Get("account1", "user1")
		assert.Equal(t, "newtoken", token)
	})

	t.Run("multiple users", func(t *testing.T) {
		manager.Set("acc", "user1", "t1")
		manager.Set("acc", "user2", "t2")
		manager.Set("acc", "user3", "t3")

		assert.Equal(t, "t1", manager.Get("acc", "user1"))
		assert.Equal(t, "t2", manager.Get("acc", "user2"))
		assert.Equal(t, "t3", manager.Get("acc", "user3"))
	})

	t.Run("empty account id", func(t *testing.T) {
		manager.Set("", "user1", "token_empty")
		token := manager.Get("", "user1")
		assert.Equal(t, "token_empty", token)
	})
}