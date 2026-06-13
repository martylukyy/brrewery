package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/api"
	appsdomain "github.com/autobrr/brrewery/internal/apps"
	"github.com/autobrr/brrewery/internal/apps/ansible"
	"github.com/autobrr/brrewery/internal/apps/detect"
	"github.com/autobrr/brrewery/internal/apps/jobs"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/system"
	"github.com/autobrr/brrewery/internal/vnstat"
)

type captureRunner struct {
	mu     sync.Mutex
	called bool
	req    ansible.RunRequest
	err    error
}

func (c *captureRunner) Run(_ context.Context, req ansible.RunRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.called = true
	c.req = req
	return c.err
}

func newSysctlClient(t *testing.T, runner *captureRunner) (*http.Client, string) {
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
	appsService := appsdomain.NewServiceWithDeps(detect.NewEvaluator(), runner, jobs.NewStore())

	srv := api.NewServer(&logger, authService, session, appsService, system.NewCollector(), vnstat.NewCollector(), runner, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{Jar: jar}

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

func TestSysctlGet(t *testing.T) {
	t.Parallel()

	client, baseURL := newSysctlClient(t, &captureRunner{})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/api/v1/system/sysctl", nil)
	require.NoError(t, err)
	res, err := client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)

	var report system.SysctlReport
	require.NoError(t, json.NewDecoder(res.Body).Decode(&report))
	require.NotEmpty(t, report.Settings)
}

func TestSysctlApply(t *testing.T) {
	t.Parallel()

	t.Run("valid request runs playbook", func(t *testing.T) {
		t.Parallel()
		runner := &captureRunner{}
		client, baseURL := newSysctlClient(t, runner)

		res := postJSON(t, client, baseURL+"/api/v1/system/sysctl", map[string]any{
			"password": "password123",
			"values": map[string]string{
				"vm.swappiness":     "20",
				"net.core.rmem_max": "16777216",
			},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		require.True(t, runner.called, "playbook should have run")
		require.True(t,
			strings.HasSuffix(runner.req.PlaybookPath, filepath.Join("playbooks", "system", "sysctl.yml")),
			"unexpected playbook path %q", runner.req.PlaybookPath,
		)
		require.Equal(t, "password123", runner.req.ExtraVars["ansible_become_password"])
		require.Equal(t, system.SysctlConfPath, runner.req.ExtraVars["sysctl_conf_path"])
		content := runner.req.ExtraVars["sysctl_conf_content"]
		require.Contains(t, content, "vm.swappiness = 20")
		require.Contains(t, content, "net.core.rmem_max = 16777216")
	})

	t.Run("wrong password is rejected before running", func(t *testing.T) {
		t.Parallel()
		runner := &captureRunner{}
		client, baseURL := newSysctlClient(t, runner)

		res := postJSON(t, client, baseURL+"/api/v1/system/sysctl", map[string]any{
			"password": "nope",
			"values":   map[string]string{"vm.swappiness": "20"},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode)
		require.False(t, runner.called, "playbook must not run on a bad password")
	})

	t.Run("invalid value is rejected", func(t *testing.T) {
		t.Parallel()
		runner := &captureRunner{}
		client, baseURL := newSysctlClient(t, runner)

		res := postJSON(t, client, baseURL+"/api/v1/system/sysctl", map[string]any{
			"password": "password123",
			"values":   map[string]string{"vm.swappiness": "9999"},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.False(t, runner.called)
	})

	t.Run("unknown key is rejected", func(t *testing.T) {
		t.Parallel()
		runner := &captureRunner{}
		client, baseURL := newSysctlClient(t, runner)

		res := postJSON(t, client, baseURL+"/api/v1/system/sysctl", map[string]any{
			"password": "password123",
			"values":   map[string]string{"kernel.shmmax": "1"},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.False(t, runner.called)
	})

	t.Run("missing password is rejected", func(t *testing.T) {
		t.Parallel()
		runner := &captureRunner{}
		client, baseURL := newSysctlClient(t, runner)

		res := postJSON(t, client, baseURL+"/api/v1/system/sysctl", map[string]any{
			"values": map[string]string{"vm.swappiness": "20"},
		})
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.False(t, runner.called)
	})
}
