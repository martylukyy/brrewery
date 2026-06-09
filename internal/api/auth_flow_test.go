package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"

	"github.com/autobrr/brrewery/internal/api"
	"github.com/autobrr/brrewery/internal/auth"
	pkgdomain "github.com/autobrr/brrewery/internal/packages"
	"github.com/autobrr/brrewery/internal/system"
	"github.com/autobrr/brrewery/internal/vnstat"
)

func TestLoginThenVersionWithSessionCookie(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := auth.NewFileStore(filepath.Join(dir, "users.json"))
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := store.CreateAdmin(auth.User{
		ID:           "admin-1",
		Username:     "admin",
		PasswordHash: hash,
	}); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	session := auth.NewSessionManager(nil)
	authService := auth.NewService(store, session)
	logger := zerolog.New(io.Discard)
	srv := api.NewServer(
		&logger,
		authService,
		session,
		pkgdomain.NewService(),
		system.NewCollector(),
		vnstat.NewCollector(),
		nil,
	)

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	baseURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}

	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "password123",
	})
	loginReq, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/auth/login", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login req: %v", err)
	}
	loginReq.Header.Set("Content-Type", "application/json")

	loginRes, err := client.Do(loginReq)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer loginRes.Body.Close()

	if loginRes.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", loginRes.StatusCode)
	}

	cookies := jar.Cookies(baseURL)
	if len(cookies) == 0 {
		t.Fatalf("no cookies in jar after login, set-cookie: %v", loginRes.Header["Set-Cookie"])
	}
	var found bool
	for _, c := range cookies {
		if c.Name == "brrewery_session" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("session cookie %q not set after login, cookies: %v", "brrewery_session", cookies)
	}

	versionReq, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/version", nil)
	if err != nil {
		t.Fatalf("version req: %v", err)
	}

	versionRes, err := client.Do(versionReq)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	defer versionRes.Body.Close()

	if versionRes.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(versionRes.Body)
		t.Fatalf("version status = %d body = %s", versionRes.StatusCode, body)
	}
}
