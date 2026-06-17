package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/api"
	appsdomain "github.com/autobrr/brrewery/internal/apps"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/system"
	"github.com/autobrr/brrewery/internal/vnstat"
	"github.com/autobrr/brrewery/internal/web"
)

// newSPATestServer builds a server with the real embedded frontend mounted so
// the SPA catch-all route is wired up the same way it is in production.
func newSPATestServer(t *testing.T) *httptest.Server {
	t.Helper()

	store := auth.NewFileStore(filepath.Join(t.TempDir(), "users.json"))
	session := auth.NewSessionManager(nil)
	authService := auth.NewService(store, session)
	logger := zerolog.New(io.Discard)

	dist, err := web.DistFS()
	require.NoError(t, err)

	srv := api.NewServer(
		&logger,
		authService,
		session,
		appsdomain.NewService(),
		system.NewCollector(),
		vnstat.NewCollector(),
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

// An unknown front-end path must be a real 404 with the HTML error page, not the
// dashboard served with a 200.
func TestSPACatchAll_UnknownPathIs404(t *testing.T) {
	ts := newSPATestServer(t)

	res, body := get(t, ts.URL+"/notvalid/")
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "text/html")
	assert.Contains(t, body, "Page not found")
}

func TestSPACatchAll_RootServesApp(t *testing.T) {
	ts := newSPATestServer(t)

	res, _ := get(t, ts.URL+"/")
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "text/html")
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
