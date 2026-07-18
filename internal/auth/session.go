package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"

	"github.com/autobrr/brrewery/internal/buildinfo"
	"github.com/autobrr/brrewery/internal/paths"
)

const sessionSecretSize = 32

func LoadOrCreateSessionSecret(path string) ([]byte, error) {
	if path == "" {
		path = paths.SessionSecretPath
	}

	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) >= sessionSecretSize {
			return data[:sessionSecretSize], nil
		}
	}

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read session secret: %w", err)
	}

	secret := make([]byte, sessionSecretSize)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate session secret: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create session secret dir: %w", err)
	}
	if err := os.WriteFile(path, secret, 0o600); err != nil {
		return nil, fmt.Errorf("write session secret: %w", err)
	}

	return secret, nil
}

func NewSessionManager(secret []byte) *scs.SessionManager {
	manager := scs.New()
	manager.Lifetime = 365 * 24 * time.Hour
	manager.Cookie.Name = "brrewery_session"
	manager.Cookie.HttpOnly = true
	manager.Cookie.SameSite = http.SameSiteLaxMode
	// The session cookie is a long-lived, root-equivalent credential, so it must
	// only ever travel over TLS. Production always serves the dashboard over
	// HTTPS (nginx terminates TLS in front of the 127.0.0.1 backend), so Secure
	// is on. It is relaxed only for local development, where the app is served
	// over plain HTTP and a Secure cookie would never be stored by the browser.
	manager.Cookie.Secure = secureCookies()
	// Persist defaults to true, which forces every cookie to carry the full
	// Lifetime expiry. Disable it so persistence is decided per login via the
	// "Remember me" choice: RememberMe(true) keeps the long-lived cookie, while
	// RememberMe(false) leaves a session-only cookie cleared when the browser closes.
	manager.Cookie.Persist = false
	manager.Codec = scs.GobCodec{}
	_ = secret // reserved for persistent session signing (M2)
	return manager
}

// secureCookies reports whether the session cookie must carry the Secure
// attribute. It is on for every real (release) build and off only for dev
// builds, which are served over plain HTTP. BRREWERY_INSECURE_COOKIES=1 forces
// it off as an explicit escape hatch for non-standard local setups; there is no
// way to force it on beyond shipping a release build, so production cannot
// accidentally end up with an insecure cookie.
func secureCookies() bool {
	if v := strings.TrimSpace(os.Getenv("BRREWERY_INSECURE_COOKIES")); v == "1" || strings.EqualFold(v, "true") {
		return false
	}
	return !isDevBuild(buildinfo.Version)
}

// isDevBuild mirrors selfupdate.IsDevBuild without importing it: a binary built
// without a release version (make dev / go build without ldflags) is a dev
// build. Kept in sync with internal/selfupdate/checker.go.
func isDevBuild(version string) bool {
	return version == "" || version == "0.0.0-dev"
}

func SessionKey() string { return "authenticated" }

func EncodeSecret(secret []byte) string {
	return base64.StdEncoding.EncodeToString(secret)
}
