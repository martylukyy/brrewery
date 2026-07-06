package deluge

import (
	"context"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // Deluge WebUI auth (web.conf pwd_sha1) is SHA1; interop, not our choice
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/autobrr/brrewery/internal/apps/extravars"
)

// EnrichAnsibleVars resolves the concrete Deluge release for the chosen version
// line and the libtorrent branch to build against, writing them into the Ansible
// extra vars. Only the Deluge release is resolved from upstream; the rest of the
// per-line build profile (Python runtime, C++ standard, setuptools pin, compiler
// flags, legacy toolchain, and the libtorrent tag + Boost pinned per branch) is
// read by the deluge_build role from the vendored manifest directly, and the
// pinned libtorrent tag is cloned rather than the branch head.
func EnrichAnsibleVars(ctx context.Context, vars map[string]string, resolver *ReleaseResolver) error {
	if resolver == nil {
		resolver = DefaultReleaseResolver()
	}

	version := strings.TrimSpace(vars[extravars.DelugeVersion])
	if version == "" {
		return errors.New("deluge_version is required")
	}

	m, err := LoadManifest()
	if err != nil {
		return err
	}
	line, err := m.ResolveSelection(version)
	if err != nil {
		return err
	}
	// Normalize to the manifest's canonical line label.
	vars[extravars.DelugeVersion] = line.Version

	branch := strings.TrimSpace(vars[extravars.LibtorrentBranch])
	if branch == "" {
		branch = line.Libtorrent.Default
	}
	if !line.AllowsBranch(branch) {
		return ErrBranchNotAllowed
	}
	vars[extravars.LibtorrentBranch] = branch

	release, err := resolver.ResolveLatest(ctx, line.Series)
	if err != nil {
		return err
	}
	vars[extravars.DelugeRelease] = release

	// Deluge's WebUI ships a default password ("deluge"); hash the brrewery user's
	// password into web.conf's pwd_salt/pwd_sha1 here so the install never leaves
	// the default in place. Computing it in Go (and dropping the plaintext) keeps
	// brrewery_user_password out of the Ansible extra vars, matching qBittorrent.
	pw := vars[extravars.BrreweryUserPassword]
	if pw == "" {
		return errors.New("brrewery_user_password is required for Deluge WebUI credentials")
	}
	salt, digest, err := delugeWebUIPasswordHash(pw)
	if err != nil {
		return fmt.Errorf("hash Deluge WebUI password: %w", err)
	}
	vars[extravars.DelugeWebUIPasswordSalt] = salt
	vars[extravars.DelugeWebUIPasswordSha1] = digest
	delete(vars, extravars.BrreweryUserPassword)
	return nil
}

// delugeWebUIPasswordHash replicates Deluge's WebUI authentication
// (deluge/ui/web/auth.py): the stored credential is pwd_sha1 = sha1(pwd_salt +
// password), where the digest is fed the salt bytes first and the UTF-8 password
// bytes second, and pwd_salt is a 40-character lowercase hex string (Deluge
// derives it from os.urandom; only the value, not its derivation, matters for
// verification). deluge-web's check_password rejects the login unless this digest
// matches exactly, so these steps must not change. The shipped default password
// "deluge" is pinned as a test vector in password_test.go.
func delugeWebUIPasswordHash(password string) (salt, digest string, err error) {
	raw := make([]byte, sha1.Size)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	salt = hex.EncodeToString(raw)
	sum := sha1.New() //nolint:gosec // must match Deluge's SHA1 WebUI auth scheme
	_, _ = sum.Write([]byte(salt))
	_, _ = sum.Write([]byte(password))
	return salt, hex.EncodeToString(sum.Sum(nil)), nil
}
