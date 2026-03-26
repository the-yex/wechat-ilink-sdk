package contextmgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextTokenManager(t *testing.T) {
	manager := NewContextTokenManager()

	t.Run("set and get", func(t *testing.T) {
		manager.Set("user1", "token123")

		token := manager.Get("user1")
		assert.Equal(t, "token123", token)
	})

	t.Run("get non-existent", func(t *testing.T) {
		token := manager.Get("nonexistent-user")
		assert.Equal(t, "", token)
	})

	t.Run("delete", func(t *testing.T) {
		manager.Set("user2", "token456")
		manager.Delete("user2")

		token := manager.Get("user2")
		assert.Equal(t, "", token)
	})

	t.Run("overwrite", func(t *testing.T) {
		manager.Set("user1", "newtoken")
		token := manager.Get("user1")
		assert.Equal(t, "newtoken", token)
	})

	t.Run("multiple users", func(t *testing.T) {
		manager.Set("user1", "t1")
		manager.Set("user2", "t2")
		manager.Set("user3", "t3")

		assert.Equal(t, "t1", manager.Get("user1"))
		assert.Equal(t, "t2", manager.Get("user2"))
		assert.Equal(t, "t3", manager.Get("user3"))
	})

	t.Run("empty user id", func(t *testing.T) {
		manager.Set("", "token_empty")
		token := manager.Get("")
		assert.Equal(t, "token_empty", token)
	})
}
