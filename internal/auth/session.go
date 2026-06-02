package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alexedwards/scs/v2"

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
	manager.Lifetime = 24 * time.Hour
	manager.Cookie.HttpOnly = true
	manager.Cookie.SameSite = http.SameSiteLaxMode
	manager.Cookie.Secure = false
	manager.Codec = scs.GobCodec{}
	_ = secret // reserved for persistent session signing (M2)
	return manager
}

func SessionKey() string { return "authenticated" }

func EncodeSecret(secret []byte) string {
	return base64.StdEncoding.EncodeToString(secret)
}
