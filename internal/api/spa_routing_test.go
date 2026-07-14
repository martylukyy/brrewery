package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/api"
	appsdomain "github.com/autobrr/brrewery/internal/apps"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/system"
	"github.com/autobrr/brrewery/internal/vnstat"
)

// newSPATestServer builds a server with a fixture frontend bundle mounted so
// the SPA catch-all route is wired up the same way it is in production. The
// fixture mirrors the real Vite output (index.html carrying the TanStack app
// mount point <div id="root">) without depending on the frontend build.
func newSPATestServer(t *testing.T) *httptest.Server {
	t.Helper()

	store := auth.NewFileStore(filepath.Join(t.TempDir(), "users.json"))
	session := auth.NewSessionManager(nil)
	authService := auth.NewService(store, session)
	logger := zerolog.New(io.Discard)

	dist := fstest.MapFS{
		"index.html": {Data: []byte(`<!doctype html><html><body><div id="root"></div></body></html>`)},
	}

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
		dist,
	)

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	return res, string(body)
}

// An unknown front-end path must be a real 404 that still serves the SPA shell,
// so the in-app React 404 page renders (not the dashboard with a 200).
func TestSPACatchAll_UnknownPathIs404(t *testing.T) {
	ts := newSPATestServer(t)

	res, body := get(t, ts.URL+"/notvalid/")
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "text/html")
	assert.Contains(t, body, `id="root"`)
}

func TestSPACatchAll_KnownRoutesServeApp(t *testing.T) {
	ts := newSPATestServer(t)

	for _, route := range []string{"/", "/login"} {
		res, body := get(t, ts.URL+route)
		assert.Equal(t, http.StatusOK, res.StatusCode, route)
		assert.Contains(t, res.Header.Get("Content-Type"), "text/html", route)
		assert.Contains(t, body, `id="root"`, route)
	}
}

// HEAD on an unknown path must also be a 404 (not a 405) — the catch-all is
// registered for HEAD as well as GET.
func TestSPACatchAll_UnknownPathHeadIs404(t *testing.T) {
	ts := newSPATestServer(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodHead, ts.URL+"/notvalid/", http.NoBody)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

// Unknown /api/ paths stay JSON 404s and never serve the HTML error page.
func TestSPACatchAll_UnknownAPIPathIsJSON(t *testing.T) {
	ts := newSPATestServer(t)

	res, body := get(t, ts.URL+"/api/does-not-exist")
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "application/json")
	assert.Contains(t, body, `"error"`)
	assert.NotContains(t, body, "Page not found")
}
