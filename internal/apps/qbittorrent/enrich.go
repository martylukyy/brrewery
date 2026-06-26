package qbittorrent

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"

	"github.com/autobrr/brrewery/internal/apps/extravars"
)

// EnrichAnsibleVars resolves the qBittorrent patch release from GitHub and
// computes the WebUI password hash, writing both into the Ansible extra vars.
//
// The build-dependency versions (Qt, zlib, Boost, OpenSSL, libtorrent) are not
// set here: they are pinned per line in the vendored manifest, which the Ansible
// build role reads directly. Only the qBittorrent patch is resolved from
// upstream, so builds stay reproducible.
func EnrichAnsibleVars(
	ctx context.Context,
	vars map[string]string,
	releaseResolver *ReleaseResolver,
) error {
	if releaseResolver == nil {
		releaseResolver = DefaultReleaseResolver()
	}

	version := strings.TrimSpace(vars[extravars.QbittorrentVersion])
	if version == "" {
		return errors.New("qbittorrent_version is required")
	}

	m, err := LoadManifest()
	if err != nil {
		return err
	}
	line, err := m.ResolveSelection(version)
	if err != nil {
		return err
	}
	vars[extravars.QbittorrentVersion] = line.Version

	patch, err := releaseResolver.ResolveLatest(ctx, line.Version)
	if err != nil {
		return err
	}
	vars[extravars.QbittorrentRelease] = patch

	pw := vars[extravars.BrreweryUserPassword]
	if pw == "" {
		return errors.New("brrewery_user_password is required for qBittorrent WebUI credentials")
	}
	hash, err := qbtWebUIPasswordHash(pw)
	if err != nil {
		return fmt.Errorf("hash qBittorrent WebUI password: %w", err)
	}
	vars[extravars.QbittorrentWebUIPasswordHash] = hash
	delete(vars, extravars.BrreweryUserPassword)

	return nil
}

// qBittorrent's WebUI authentication (Utils::Password::PBKDF2 in
// src/base/utils/password.cpp) verifies the stored credential by recomputing
// PBKDF2-HMAC-SHA512 over the entered password with the stored salt. These
// parameters must match it exactly or the WebUI login rejects the password:
// a 16-byte salt, 100000 iterations and a 64-byte derived key, serialized as
// @ByteArray(<salt_b64>:<hash_b64>).
const (
	qbtPBKDF2SaltLen    = 16
	qbtPBKDF2Iterations = 100000
	qbtPBKDF2KeyLen     = 64
)

// qbtWebUIPasswordHash returns a qBittorrent-compatible PBKDF2-HMAC-SHA512 hash
// in the @ByteArray(<salt_b64>:<hash_b64>) format written to qBittorrent.conf.
func qbtWebUIPasswordHash(password string) (string, error) {
	salt := make([]byte, qbtPBKDF2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := pbkdf2.Key([]byte(password), salt, qbtPBKDF2Iterations, qbtPBKDF2KeyLen, sha512.New)
	return "@ByteArray(" + base64.StdEncoding.EncodeToString(salt) + ":" + base64.StdEncoding.EncodeToString(key) + ")", nil
}
