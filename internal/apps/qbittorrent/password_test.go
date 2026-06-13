package qbittorrent

import (
	"crypto/sha512"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/crypto/pbkdf2"
)

// decodeQbtHash parses the @ByteArray(<salt_b64>:<hash_b64>) serialization that
// qBittorrent stores in WebUI\Password_PBKDF2 and returns the raw salt and hash.
func decodeQbtHash(t *testing.T, encoded string) (salt, hash []byte) {
	t.Helper()
	require.True(t, strings.HasPrefix(encoded, "@ByteArray("), "missing @ByteArray prefix: %q", encoded)
	require.True(t, strings.HasSuffix(encoded, ")"), "missing closing paren: %q", encoded)
	inner := strings.TrimSuffix(strings.TrimPrefix(encoded, "@ByteArray("), ")")
	parts := strings.Split(inner, ":")
	require.Len(t, parts, 2, "expected <salt>:<hash>, got %q", inner)
	salt, err := base64.StdEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	hash, err = base64.StdEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	return salt, hash
}

// TestQbtWebUIPasswordHash_matchesQbittorrentVerification recomputes the hash the
// way qBittorrent's Utils::Password::PBKDF2::verify does (PBKDF2-HMAC-SHA512, 100000
// iterations, 64-byte key over the stored salt) and asserts it matches. The literals
// here intentionally pin the on-disk contract: if the generator regresses to a
// different digest or key length the WebUI login breaks, and so does this test.
func TestQbtWebUIPasswordHash_matchesQbittorrentVerification(t *testing.T) {
	t.Parallel()

	const password = "Tr0ub4dor&3 with spaces"

	encoded, err := qbtWebUIPasswordHash(password)
	require.NoError(t, err)

	salt, hash := decodeQbtHash(t, encoded)
	assert.Len(t, salt, 16, "qBittorrent uses a 16-byte salt")
	assert.Len(t, hash, 64, "qBittorrent uses a 64-byte (SHA-512) derived key")

	want := pbkdf2.Key([]byte(password), salt, 100000, 64, sha512.New)
	assert.Equal(t, want, hash, "stored hash must match PBKDF2-HMAC-SHA512 verification")

	// A different password must not verify against the same salt.
	wrong := pbkdf2.Key([]byte("not the password"), salt, 100000, 64, sha512.New)
	assert.NotEqual(t, hash, wrong)
}

// TestQbtWebUIPasswordHash_saltIsRandom guards against a fixed salt, which would
// make all installs share a hash and leak whether two users picked the same password.
func TestQbtWebUIPasswordHash_saltIsRandom(t *testing.T) {
	t.Parallel()

	first, err := qbtWebUIPasswordHash("same-password")
	require.NoError(t, err)
	second, err := qbtWebUIPasswordHash("same-password")
	require.NoError(t, err)

	assert.NotEqual(t, first, second, "each hash must use a fresh random salt")
}
