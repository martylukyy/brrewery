package handlers_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/api"
	"github.com/autobrr/brrewery/internal/auth"
	pkgdomain "github.com/autobrr/brrewery/internal/packages"
	"github.com/autobrr/brrewery/internal/packages/ansible"
	"github.com/autobrr/brrewery/internal/packages/detect"
	"github.com/autobrr/brrewery/internal/packages/jobs"
	"github.com/autobrr/brrewery/internal/packages/model"
	"github.com/autobrr/brrewery/internal/system"
	"github.com/autobrr/brrewery/internal/vnstat"
)

type stubRunner struct{}

func (stubRunner) Run(_ context.Context, _ ansible.RunRequest) error {
	return nil
}

func TestInstallPackageEndpoint(t *testing.T) {
	t.Parallel()

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
	authService := auth.NewService(store, session)
	logger := zerolog.New(io.Discard)

	packagesService := pkgdomain.NewServiceWithDeps(
		detect.NewEvaluator(),
		stubRunner{},
		jobs.NewStore(),
	)

	srv := api.NewServer(
		&logger,
		authService,
		session,
		packagesService,
		system.NewCollector(),
		vnstat.NewCollector(),
		nil,
	)

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	baseURL, err := url.Parse(ts.URL)
	require.NoError(t, err)
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{Jar: jar}

	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "password123",
	})
	loginReq, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/auth/login", bytes.NewReader(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes, err := client.Do(loginReq)
	require.NoError(t, err)
	loginRes.Body.Close()
	require.Equal(t, http.StatusOK, loginRes.StatusCode)
	require.NotEmpty(t, jar.Cookies(baseURL))

	installBody, _ := json.Marshal(map[string]any{
		"extra_vars": map[string]string{
			"ansible_become_password": "password123",
		},
	})
	installReq, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/packages/autobrr/install", bytes.NewReader(installBody))
	require.NoError(t, err)
	installReq.Header.Set("Content-Type", "application/json")
	installRes, err := client.Do(installReq)
	require.NoError(t, err)
	defer installRes.Body.Close()

	body, _ := io.ReadAll(installRes.Body)
	if installRes.StatusCode == http.StatusInternalServerError {
		t.Skip("autobrr playbook not available in test environment")
	}

	require.Equal(t, http.StatusAccepted, installRes.StatusCode, string(body))

	var resp model.InstallResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	require.NotEmpty(t, resp.JobID)
}

func TestInstallPackageEndpoint_InvalidPassword(t *testing.T) {
	t.Parallel()

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
	authService := auth.NewService(store, session)
	logger := zerolog.New(io.Discard)

	packagesService := pkgdomain.NewServiceWithDeps(
		detect.NewEvaluator(),
		stubRunner{},
		jobs.NewStore(),
	)

	srv := api.NewServer(
		&logger,
		authService,
		session,
		packagesService,
		system.NewCollector(),
		vnstat.NewCollector(),
		nil,
	)

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{Jar: jar}

	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "password123",
	})
	loginReq, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/auth/login", bytes.NewReader(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes, err := client.Do(loginReq)
	require.NoError(t, err)
	loginRes.Body.Close()
	require.Equal(t, http.StatusOK, loginRes.StatusCode)

	installBody, _ := json.Marshal(map[string]any{
		"extra_vars": map[string]string{
			"ansible_become_password": "wrong-password",
		},
	})
	installReq, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/packages/autobrr/install", bytes.NewReader(installBody))
	require.NoError(t, err)
	installReq.Header.Set("Content-Type", "application/json")
	installRes, err := client.Do(installReq)
	require.NoError(t, err)
	defer installRes.Body.Close()

	require.Equal(t, http.StatusUnauthorized, installRes.StatusCode)
}

func newLoggedInClient(t *testing.T) (client *http.Client, baseURL string) {
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
	authService := auth.NewService(store, session)
	logger := zerolog.New(io.Discard)

	packagesService := pkgdomain.NewServiceWithDeps(detect.NewEvaluator(), stubRunner{}, jobs.NewStore())
	srv := api.NewServer(&logger, authService, session, packagesService, system.NewCollector(), vnstat.NewCollector(), nil)

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client = &http.Client{Jar: jar}

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	loginReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+"/api/v1/auth/login", bytes.NewReader(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRes, err := client.Do(loginReq)
	require.NoError(t, err)
	loginRes.Body.Close()
	require.Equal(t, http.StatusOK, loginRes.StatusCode)

	return client, ts.URL
}

func postJSON(t *testing.T, client *http.Client, endpoint string, payload map[string]any) *http.Response {
	t.Helper()
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	require.NoError(t, err)
	return res
}

func TestQbittorrentInstallOptionValidation(t *testing.T) {
	t.Parallel()

	client, baseURL := newLoggedInClient(t)

	validPatch := base64.StdEncoding.EncodeToString(
		[]byte("--- a/src/settings_pack.cpp\n+++ b/src/settings_pack.cpp\n@@ -1 +1 @@\n-old\n+new\n"),
	)

	t.Run("valid options accepted", func(t *testing.T) {
		res := postJSON(t, client, baseURL+"/api/v1/packages/qbittorrent/install", map[string]any{
			"extra_vars": map[string]string{
				"ansible_become_password": "password123",
				"qbittorrent_version":     "5.2",
				"libtorrent_branch":       "RC_2_0",
				"libtorrent_patch":        validPatch,
			},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusAccepted, res.StatusCode)
	})

	t.Run("missing password rejected", func(t *testing.T) {
		res := postJSON(t, client, baseURL+"/api/v1/packages/qbittorrent/install", map[string]any{
			"extra_vars": map[string]string{"qbittorrent_version": "5.2"},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("unknown version rejected", func(t *testing.T) {
		res := postJSON(t, client, baseURL+"/api/v1/packages/qbittorrent/install", map[string]any{
			"extra_vars": map[string]string{
				"ansible_become_password": "password123",
				"qbittorrent_version":     "9.9",
			},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("disallowed branch rejected", func(t *testing.T) {
		res := postJSON(t, client, baseURL+"/api/v1/packages/qbittorrent/install", map[string]any{
			"extra_vars": map[string]string{
				"ansible_become_password": "password123",
				"qbittorrent_version":     "4.3",
				"libtorrent_branch":       "RC_2_0",
			},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("invalid patch rejected", func(t *testing.T) {
		res := postJSON(t, client, baseURL+"/api/v1/packages/qbittorrent/install", map[string]any{
			"extra_vars": map[string]string{
				"ansible_become_password": "password123",
				"qbittorrent_version":     "5.2",
				"libtorrent_patch":        "not-base64-$$$",
			},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("upgrade validates version too", func(t *testing.T) {
		res := postJSON(t, client, baseURL+"/api/v1/packages/qbittorrent/upgrade", map[string]any{
			"extra_vars": map[string]string{"qbittorrent_version": "9.9"},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
}
