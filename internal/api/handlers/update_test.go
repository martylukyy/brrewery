package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/api"
	appsdomain "github.com/autobrr/brrewery/internal/apps"
	"github.com/autobrr/brrewery/internal/apps/detect"
	"github.com/autobrr/brrewery/internal/apps/jobs"
	"github.com/autobrr/brrewery/internal/apps/model"
	"github.com/autobrr/brrewery/internal/selfupdate"
	"github.com/autobrr/brrewery/internal/system"
	"github.com/autobrr/brrewery/internal/vnstat"
)

type fakeUpdateChecker struct {
	mu        sync.Mutex
	status    selfupdate.Status
	refreshed bool
}

func (f *fakeUpdateChecker) Status() selfupdate.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *fakeUpdateChecker) Refresh(_ context.Context) (selfupdate.Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshed = true
	return f.status, nil
}

type fakeUpdateStarter struct {
	mu     sync.Mutex
	job    model.Job
	err    error
	called bool
}

func (f *fakeUpdateStarter) Start(_ context.Context) (model.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called = true
	return f.job, f.err
}

func (f *fakeUpdateStarter) wasCalled() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.called
}

func newUpdateClient(t *testing.T, checker *fakeUpdateChecker, starter *fakeUpdateStarter) (client *http.Client, baseURL string) {
	t.Helper()

	authService, session := newAdminAuthService(t)
	logger := zerolog.New(io.Discard)
	appsService := appsdomain.NewServiceWithDeps(detect.NewEvaluator(), nil, jobs.NewStore())

	srv := api.NewServer(&logger, authService, session, appsService, system.NewCollector(), vnstat.NewCollector(), nil, checker, starter, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	return loginAsAdmin(t, ts.URL), ts.URL
}

func availableStatus() selfupdate.Status {
	now := time.Now().UTC()
	return selfupdate.Status{
		CurrentVersion:  "1.0.0",
		LatestVersion:   "1.1.0",
		LatestTag:       "v1.1.0",
		UpdateAvailable: true,
		CheckedAt:       &now,
	}
}

func TestUpdateStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns cached status", func(t *testing.T) {
		t.Parallel()
		checker := &fakeUpdateChecker{status: availableStatus()}
		client, baseURL := newUpdateClient(t, checker, &fakeUpdateStarter{})

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/api/v1/update", http.NoBody)
		require.NoError(t, err)
		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var status selfupdate.Status
		require.NoError(t, json.NewDecoder(res.Body).Decode(&status))
		require.True(t, status.UpdateAvailable)
		require.Equal(t, "1.1.0", status.LatestVersion)
		require.False(t, checker.refreshed, "plain GET must serve the cache")
	})

	t.Run("refresh=1 checks GitHub first", func(t *testing.T) {
		t.Parallel()
		checker := &fakeUpdateChecker{status: availableStatus()}
		client, baseURL := newUpdateClient(t, checker, &fakeUpdateStarter{})

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/api/v1/update?refresh=1", http.NoBody)
		require.NoError(t, err)
		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.True(t, checker.refreshed)
	})

	t.Run("requires auth", func(t *testing.T) {
		t.Parallel()
		checker := &fakeUpdateChecker{status: availableStatus()}
		_, baseURL := newUpdateClient(t, checker, &fakeUpdateStarter{})

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/api/v1/update", http.NoBody)
		require.NoError(t, err)
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})
}

func TestUpdateStart(t *testing.T) {
	t.Parallel()

	t.Run("starts the update job", func(t *testing.T) {
		t.Parallel()
		starter := &fakeUpdateStarter{job: model.Job{ID: "job-123"}}
		client, baseURL := newUpdateClient(t, &fakeUpdateChecker{status: availableStatus()}, starter)

		res := postJSON(t, client, baseURL+"/api/v1/update", map[string]any{"password": "password123"})
		defer res.Body.Close()
		require.Equal(t, http.StatusAccepted, res.StatusCode)

		var body model.InstallResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
		require.Equal(t, "job-123", body.JobID)
		require.True(t, starter.wasCalled())
	})

	t.Run("missing password is rejected", func(t *testing.T) {
		t.Parallel()
		starter := &fakeUpdateStarter{}
		client, baseURL := newUpdateClient(t, &fakeUpdateChecker{}, starter)

		res := postJSON(t, client, baseURL+"/api/v1/update", map[string]any{})
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.False(t, starter.wasCalled())
	})

	t.Run("wrong password is rejected", func(t *testing.T) {
		t.Parallel()
		starter := &fakeUpdateStarter{}
		client, baseURL := newUpdateClient(t, &fakeUpdateChecker{}, starter)

		res := postJSON(t, client, baseURL+"/api/v1/update", map[string]any{"password": "nope"})
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode)
		require.False(t, starter.wasCalled())
	})

	t.Run("update in progress conflicts", func(t *testing.T) {
		t.Parallel()
		starter := &fakeUpdateStarter{err: selfupdate.ErrUpdateInProgress}
		client, baseURL := newUpdateClient(t, &fakeUpdateChecker{}, starter)

		res := postJSON(t, client, baseURL+"/api/v1/update", map[string]any{"password": "password123"})
		defer res.Body.Close()
		require.Equal(t, http.StatusConflict, res.StatusCode)
	})

	t.Run("no update available conflicts", func(t *testing.T) {
		t.Parallel()
		starter := &fakeUpdateStarter{err: selfupdate.ErrNoUpdate}
		client, baseURL := newUpdateClient(t, &fakeUpdateChecker{}, starter)

		res := postJSON(t, client, baseURL+"/api/v1/update", map[string]any{"password": "password123"})
		defer res.Body.Close()
		require.Equal(t, http.StatusConflict, res.StatusCode)
	})

	t.Run("unsupported install returns 501", func(t *testing.T) {
		t.Parallel()
		starter := &fakeUpdateStarter{err: selfupdate.ErrUnsupported}
		client, baseURL := newUpdateClient(t, &fakeUpdateChecker{}, starter)

		res := postJSON(t, client, baseURL+"/api/v1/update", map[string]any{"password": "password123"})
		defer res.Body.Close()
		require.Equal(t, http.StatusNotImplemented, res.StatusCode)
	})
}
