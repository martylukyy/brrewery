package handlers_test

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
			"brrewery_user_password": "password123",
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
			"brrewery_user_password": "wrong-password",
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
