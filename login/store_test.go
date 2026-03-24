package login

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryTokenStore(t *testing.T) {
	store := NewMemoryTokenStore()

	t.Run("save and load", func(t *testing.T) {
		token := &TokenInfo{
			Token:   "test-token",
			BaseURL: "https://example.com",
			UserID:  "user123",
		}

		err := store.Save("account1", token)
		require.NoError(t, err)

		loaded, err := store.Load("account1")
		require.NoError(t, err)
		assert.Equal(t, token, loaded)
	})

	t.Run("load non-existent", func(t *testing.T) {
		loaded, err := store.Load("nonexistent")
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("delete", func(t *testing.T) {
		err := store.Delete("account1")
		require.NoError(t, err)

		loaded, err := store.Load("account1")
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("list", func(t *testing.T) {
		_ = store.Save("acc1", &TokenInfo{Token: "t1"})
		_ = store.Save("acc2", &TokenInfo{Token: "t2"})

		accounts, err := store.List()
		require.NoError(t, err)
		assert.Len(t, accounts, 2)
		assert.Contains(t, accounts, "acc1")
		assert.Contains(t, accounts, "acc2")
	})
}

func TestFileTokenStore(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	store, err := NewFileTokenStore(tempDir)
	require.NoError(t, err)

	t.Run("save and load", func(t *testing.T) {
		token := &TokenInfo{
			Token:   "test-token",
			BaseURL: "https://example.com",
			UserID:  "user123",
		}

		err := store.Save("account1", token)
		require.NoError(t, err)

		// Verify file exists
		path := filepath.Join(tempDir, "account1.json")
		_, err = os.Stat(path)
		require.NoError(t, err)

		// Verify file permissions
		info, _ := os.Stat(path)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

		loaded, err := store.Load("account1")
		require.NoError(t, err)
		assert.Equal(t, token, loaded)
	})

	t.Run("load non-existent", func(t *testing.T) {
		loaded, err := store.Load("nonexistent")
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("delete", func(t *testing.T) {
		err := store.Delete("account1")
		require.NoError(t, err)

		loaded, err := store.Load("account1")
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("list", func(t *testing.T) {
		_ = store.Save("acc1", &TokenInfo{Token: "t1"})
		_ = store.Save("acc2", &TokenInfo{Token: "t2"})

		accounts, err := store.List()
		require.NoError(t, err)
		assert.Len(t, accounts, 2)
	})
}

func TestFileTokenStore_DefaultDir(t *testing.T) {
	// Test with empty path (uses default home dir)
	store, err := NewFileTokenStore("")
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestFileTokenStore_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewFileTokenStore(tempDir)
	require.NoError(t, err)

	// Write invalid JSON
	path := filepath.Join(tempDir, "invalid.json")
	err = os.WriteFile(path, []byte("not json"), 0600)
	require.NoError(t, err)

	// Should return error
	_, err = store.Load("invalid")
	require.Error(t, err)
}