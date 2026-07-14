package selfupdate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestChecker(t *testing.T, currentVersion string, handler http.HandlerFunc) *Checker {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	checker := NewChecker("martylukyy/brrewery")
	checker.apiBase = server.URL
	checker.currentVersion = currentVersion
	return checker
}

func releasesHandler(t *testing.T, body string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/martylukyy/brrewery/releases", r.URL.Path)
		assert.NotEmpty(t, r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

func TestRefreshPicksFirstNonDraft(t *testing.T) {
	checker := newTestChecker(t, "1.0.0", releasesHandler(t, `[
		{"tag_name": "v1.3.0", "draft": true},
		{"tag_name": "v1.2.0-rc.1", "draft": false, "prerelease": true},
		{"tag_name": "v1.1.0", "draft": false}
	]`))

	status, err := checker.Refresh(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "v1.2.0-rc.1", status.LatestTag)
	assert.Equal(t, "1.2.0-rc.1", status.LatestVersion)
	assert.Equal(t, "1.0.0", status.CurrentVersion)
	assert.True(t, status.UpdateAvailable)
	assert.NotNil(t, status.CheckedAt)
	assert.Empty(t, status.Error)
}

func TestRefreshNoUpdateWhenCurrent(t *testing.T) {
	checker := newTestChecker(t, "1.2.0", releasesHandler(t, `[
		{"tag_name": "v1.2.0", "draft": false}
	]`))

	status, err := checker.Refresh(context.Background())
	require.NoError(t, err)
	assert.False(t, status.UpdateAvailable)
}

func TestRefreshErrorKeepsPreviousStatus(t *testing.T) {
	fail := false
	checker := newTestChecker(t, "1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		if fail {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(`[{"tag_name": "v1.1.0", "draft": false}]`))
	})

	first, err := checker.Refresh(context.Background())
	require.NoError(t, err)
	require.True(t, first.UpdateAvailable)

	fail = true
	second, err := checker.Refresh(context.Background())
	require.Error(t, err)

	assert.Equal(t, "v1.1.0", second.LatestTag)
	assert.True(t, second.UpdateAvailable)
	assert.NotEmpty(t, second.Error)
	assert.Equal(t, first.CheckedAt, second.CheckedAt)

	cached := checker.Status()
	assert.Equal(t, second, cached)
}

func TestRefreshNoReleases(t *testing.T) {
	checker := newTestChecker(t, "1.0.0", releasesHandler(t, `[{"tag_name": "v2.0.0", "draft": true}]`))

	_, err := checker.Refresh(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no published releases")
}

func TestStatusBeforeFirstCheck(t *testing.T) {
	checker := NewChecker("martylukyy/brrewery")
	checker.currentVersion = "1.0.0"

	status := checker.Status()
	assert.Equal(t, "1.0.0", status.CurrentVersion)
	assert.False(t, status.UpdateAvailable)
	assert.Nil(t, status.CheckedAt)
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"older patch", "1.0.0", "v1.0.1", true},
		{"older minor", "1.0.0", "v1.1.0", true},
		{"equal", "1.2.0", "v1.2.0", false},
		{"newer than latest", "1.3.0", "v1.2.0", false},
		{"prerelease newer than stable", "1.1.0", "v1.2.0-rc.1", true},
		{"stable newer than its prerelease", "1.2.0", "v1.2.0-rc.1", false},
		{"prerelease to its stable", "1.2.0-rc.1", "v1.2.0", true},
		{"dev build never updates", "0.0.0-dev", "v9.9.9", false},
		{"empty version never updates", "", "v1.0.0", false},
		{"invalid tag", "1.0.0", "vnext", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isNewer(tc.current, tc.latest))
		})
	}
}

func TestRepoFromEnv(t *testing.T) {
	t.Setenv("BRREWERY_UPDATE_REPO", "")
	t.Setenv("BRREWERY_REPO_URL", "")
	assert.Equal(t, DefaultRepo, RepoFromEnv())

	t.Setenv("BRREWERY_REPO_URL", "https://github.com/someone/fork.git")
	assert.Equal(t, "someone/fork", RepoFromEnv())

	t.Setenv("BRREWERY_UPDATE_REPO", "other/repo")
	assert.Equal(t, "other/repo", RepoFromEnv())
}
