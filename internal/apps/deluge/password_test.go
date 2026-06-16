package deluge

import (
	"crypto/sha1" //nolint:gosec // verifying Deluge's SHA1-based WebUI auth, not securing anything
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Deluge ships a default WebUI password of "deluge" whose stored credential is
// the pwd_salt/pwd_sha1 pair below (deluge/ui/web/server.py CONFIG_DEFAULTS,
// identical across the 1.3.15, 2.1.1 and 2.2.0 tags). It is the canonical proof
// that pwd_sha1 == sha1(pwd_salt + password) with the salt hashed first: if our
// hashing regresses to a different construction the WebUI login breaks, and so
// does this test.
const (
	delugeDefaultSalt = "c26ab3bbd8b137f99cd83c2c1c0963bcc1a35cad"
	delugeDefaultSha1 = "2ce1a410bcdcc53064129b6d950f2e9fee4edc1e"
)

// delugePwdSha1 recomputes the digest exactly as deluge-web's check_password
// does: sha1 over the salt bytes followed by the password bytes.
func delugePwdSha1(salt, password string) string {
	sum := sha1.New() //nolint:gosec // matches Deluge's WebUI auth scheme
	_, _ = sum.Write([]byte(salt))
	_, _ = sum.Write([]byte(password))
	return hex.EncodeToString(sum.Sum(nil))
}

// TestDelugeWebUIPasswordHash_matchesDelugeDefaultVector pins the construction
// against Deluge's own shipped default credential, independent of our generator.
func TestDelugeWebUIPasswordHash_matchesDelugeDefaultVector(t *testing.T) {
	t.Parallel()
	assert.Equal(t, delugeDefaultSha1, delugePwdSha1(delugeDefaultSalt, "deluge"),
		`sha1(pwd_salt + "deluge") must equal Deluge's shipped default pwd_sha1`)
}

// TestDelugeWebUIPasswordHash_matchesDelugeVerification recomputes our generator's
// output the way check_password does and asserts it verifies — and that a wrong
// password does not — so a stored credential we write always logs in.
func TestDelugeWebUIPasswordHash_matchesDelugeVerification(t *testing.T) {
	t.Parallel()

	const password = "Tr0ub4dor&3 with spaces"

	salt, digest, err := delugeWebUIPasswordHash(password)
	require.NoError(t, err)
	assert.Len(t, salt, 40, "Deluge pwd_salt is a 40-char hex string")
	assert.Len(t, digest, 40, "a sha1 hexdigest is 40 chars")

	assert.Equal(t, digest, delugePwdSha1(salt, password),
		"stored pwd_sha1 must verify under Deluge's check_password")
	assert.NotEqual(t, digest, delugePwdSha1(salt, "not the password"),
		"a different password must not verify against the same salt")
}

// TestDelugeWebUIPasswordHash_saltIsRandom guards against a fixed salt, which
// would make every install share a digest and leak whether two users picked the
// same password.
func TestDelugeWebUIPasswordHash_saltIsRandom(t *testing.T) {
	t.Parallel()

	firstSalt, firstDigest, err := delugeWebUIPasswordHash("same-password")
	require.NoError(t, err)
	secondSalt, secondDigest, err := delugeWebUIPasswordHash("same-password")
	require.NoError(t, err)

	assert.NotEqual(t, firstSalt, secondSalt, "each hash must use a fresh random salt")
	assert.NotEqual(t, firstDigest, secondDigest, "a fresh salt must yield a different digest")
}
