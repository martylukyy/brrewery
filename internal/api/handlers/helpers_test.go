package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"path/filepath"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/auth"
)

// newAdminAuthService builds an auth service backed by a temp user store with
// one admin account ("admin" / "password123").
func newAdminAuthService(t *testing.T) (*auth.Service, *scs.SessionManager) {
	t.Helper()

	dir := t.TempDir()
	store := auth.NewFileStore(filepath.Join(dir, "users.json"))
	hash, err := auth.HashPassword("password123")
	require.NoError(t, err)
	require.NoError(t, store.CreateAdmin(auth.User{
		ID:           "admin-1",
		Username:     "admin",
		PasswordHash: hash,
	}))

	session := auth.NewSessionManager(nil)
	return auth.NewService(store, session), session
}

// loginAsAdmin returns a cookie-jar client signed in as the admin account
// created by newAdminAuthService.
func loginAsAdmin(t *testing.T, baseURL string) *http.Client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{Jar: jar}

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	loginReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/api/v1/auth/login", bytes.NewReader(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes, err := client.Do(loginReq)
	require.NoError(t, err)
	loginRes.Body.Close()
	require.Equal(t, http.StatusOK, loginRes.StatusCode)

	return client
}
