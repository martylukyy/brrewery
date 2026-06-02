package auth

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStore_CreateAdmin(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewFileStore(filepath.Join(dir, "users.json"))

	has, err := store.HasUsers()
	require.NoError(t, err)
	assert.False(t, has)

	err = store.CreateAdmin(User{ID: "1", Username: "admin", PasswordHash: "hash"})
	require.NoError(t, err)

	has, err = store.HasUsers()
	require.NoError(t, err)
	assert.True(t, has)

	err = store.CreateAdmin(User{ID: "2", Username: "other", PasswordHash: "hash2"})
	assert.ErrorIs(t, err, ErrUserExists)
}
