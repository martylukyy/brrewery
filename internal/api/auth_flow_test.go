package api_test

import (
	"bytes"
	"context"
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
	appsdomain "github.com/autobrr/brrewery/internal/apps"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/system"
	"github.com/autobrr/brrewery/internal/vnstat"
)

// newLoginTestServer spins up an API server backed by a temp user store seeded
// with a single admin account (admin / password123).
func newLoginTestServer(t *testing.T) *httptest.Server {
	t.Helper()

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
		appsdomain.NewService(),
		system.NewCollector(),
		vnstat.NewCollector(),
		nil,
		nil,
		nil,
		nil,
	)

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// doLogin authenticates the seeded admin against the server, honouring the
// caller's rememberMe choice.
func doLogin(t *testing.T, client *http.Client, baseURL string, rememberMe bool) *http.Response {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"username":    "admin",
		"password":    "password123",
		"remember_me": rememberMe,
	})
	if err != nil {
		t.Fatalf("marshal login: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/api/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	return res
}

// sessionCookie returns the brrewery session cookie from a response, failing the
// test if it is absent.
func sessionCookie(t *testing.T, res *http.Response) *http.Cookie {
	t.Helper()

	for _, c := range res.Cookies() {
		if c.Name == "brrewery_session" {
			return c
		}
	}
	t.Fatalf("session cookie %q not set, set-cookie: %v", "brrewery_session", res.Header["Set-Cookie"])
	return nil
}

func TestLoginThenVersionWithSessionCookie(t *testing.T) {
	t.Parallel()

	ts := newLoginTestServer(t)

	baseURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}

	loginRes := doLogin(t, client, ts.URL, true)
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

	versionReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+"/api/v1/version", http.NoBody)
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

// TestLoginRememberMeControlsCookiePersistence verifies that the "Remember me"
// choice decides whether the session cookie persists for its full lifetime or is
// dropped when the browser closes.
func TestLoginRememberMeControlsCookiePersistence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		rememberMe bool
		persistent bool
	}{
		{name: "remember me sets a persistent cookie", rememberMe: true, persistent: true},
		{name: "without remember me cookie is session only", rememberMe: false, persistent: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ts := newLoginTestServer(t)
			res := doLogin(t, &http.Client{}, ts.URL, tc.rememberMe)
			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				t.Fatalf("login status = %d", res.StatusCode)
			}

			c := sessionCookie(t, res)
			if tc.persistent {
				if c.MaxAge <= 0 {
					t.Fatalf("expected persistent cookie, got MaxAge=%d Expires=%v", c.MaxAge, c.Expires)
				}
			} else if c.MaxAge != 0 || !c.Expires.IsZero() {
				t.Fatalf("expected session-only cookie, got MaxAge=%d Expires=%v", c.MaxAge, c.Expires)
			}
		})
	}
}
