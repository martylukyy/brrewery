package web_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/web"
)

// The fixture mirrors the real bundle: index.html carries the SPA mount point
// (<div id="root">), 404.html is the shared error page, plus a couple of assets.
func newTestFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte(`<!doctype html><html><body><div id="root"></div></body></html>`)},
		"404.html":      {Data: []byte(`<!doctype html><title>404</title><h1>Page not found</h1>`)},
		"assets/app.js": {Data: []byte("console.log('app')")},
		"logos/x.webp":  {Data: []byte("webp-bytes")},
	}
}

func serve(t *testing.T, h *web.Handler, method, target string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	h.ServeSPA(rec, req)
	return rec.Result()
}

func TestServeSPA_Root(t *testing.T) {
	res := serve(t, web.NewHandler(newTestFS()), http.MethodGet, "/")
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "text/html")
	assert.Contains(t, string(body), `id="root"`)
}

func TestServeSPA_StaticAssetServesWithContentType(t *testing.T) {
	cases := []struct {
		target      string
		body        string
		contentType string
	}{
		{"/assets/app.js", "console.log('app')", "javascript"},
		{"/logos/x.webp", "webp-bytes", "image/webp"},
	}
	for _, tc := range cases {
		t.Run(tc.target, func(t *testing.T) {
			res := serve(t, web.NewHandler(newTestFS()), http.MethodGet, tc.target)
			defer res.Body.Close()

			body, _ := io.ReadAll(res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, tc.body, string(body))
			assert.Contains(t, res.Header.Get("Content-Type"), tc.contentType)
		})
	}
}

func TestServeSPA_IndexHTMLPath(t *testing.T) {
	res := serve(t, web.NewHandler(newTestFS()), http.MethodGet, "/index.html")
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
}

// The core bug: unknown paths used to fall back to index.html with a 200 and
// render the dashboard. They must now be a real 404 with the error page.
func TestServeSPA_UnknownPathReturns404(t *testing.T) {
	cases := []string{
		"/notvalid/",
		"/notvalid",
		"/does/not/exist",
		"/assets/missing.js",
		"/../handler.go",       // path traversal must not escape the bundle
		"/assets/../../go.mod", // nor through an existing dir
	}
	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			res := serve(t, web.NewHandler(newTestFS()), http.MethodGet, target)
			defer res.Body.Close()

			body, _ := io.ReadAll(res.Body)
			assert.Equal(t, http.StatusNotFound, res.StatusCode)
			assert.Contains(t, res.Header.Get("Content-Type"), "text/html")
			assert.Equal(t, "no-store", res.Header.Get("Cache-Control"))
			assert.Contains(t, string(body), "Page not found")
			// It must not silently fall back to the dashboard shell.
			assert.NotContains(t, string(body), `id="root"`)
		})
	}
}

// A directory must not be served (no listing) and must not fall back to index.
func TestServeSPA_DirectoryReturns404(t *testing.T) {
	res := serve(t, web.NewHandler(newTestFS()), http.MethodGet, "/assets")
	defer res.Body.Close()

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestServeSPA_HeadHasNoBody(t *testing.T) {
	res := serve(t, web.NewHandler(newTestFS()), http.MethodHead, "/notvalid")
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Empty(t, body)
}

// When the bundle lacks 404.html (e.g. an old build), the handler still returns
// a 404 with the inline fallback rather than failing.
func TestServeSPA_NotFoundFallbackWhenPageMissing(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": {Data: []byte(`<div id="root"></div>`)},
	}
	res := serve(t, web.NewHandler(fsys), http.MethodGet, "/notvalid")
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Contains(t, string(body), "Page not found")
	assert.NotContains(t, string(body), `id="root"`)
}

func TestServeSPA_NilFSNotBuilt(t *testing.T) {
	res := serve(t, web.NewHandler(nil), http.MethodGet, "/")
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Contains(t, string(body), "Frontend not built")
}

func TestServeSPA_RealDistRootServes(t *testing.T) {
	dist, err := web.DistFS()
	require.NoError(t, err)

	res := serve(t, web.NewHandler(dist), http.MethodGet, "/")
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	res404 := serve(t, web.NewHandler(dist), http.MethodGet, "/notvalid/")
	defer res404.Body.Close()
	body, _ := io.ReadAll(res404.Body)
	assert.Equal(t, http.StatusNotFound, res404.StatusCode)
	assert.Contains(t, string(body), "Page not found")
	// Fingerprint the styled bundle page (not the inline fallback) so this
	// proves the embedded 404.html — the single source of truth — is served.
	assert.Contains(t, string(body), "Return to dashboard")
}
