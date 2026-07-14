package web_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/brrewery/internal/web"
)

// The fixture mirrors the real bundle: index.html carries the SPA mount point
// (<div id="root">), plus a couple of static assets.
func newTestFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte(`<!doctype html><html><body><div id="root"></div></body></html>`)},
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

// Known client routes serve the SPA shell with a 200 so the app boots.
func TestServeSPA_KnownRoutesServeShell(t *testing.T) {
	for _, target := range []string{"/", "/index.html", "/login"} {
		t.Run(target, func(t *testing.T) {
			res := serve(t, web.NewHandler(newTestFS()), http.MethodGet, target)
			defer res.Body.Close()

			body, _ := io.ReadAll(res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Contains(t, res.Header.Get("Content-Type"), "text/html")
			assert.Contains(t, string(body), `id="root"`)
		})
	}
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

// The core behaviour: an unknown path serves the shell but with a true 404 so
// the client renders <NotFound/>. It must not be a 200, and must not 404 with an
// empty/error body that fails to boot the app.
func TestServeSPA_UnknownPathServesShellAs404(t *testing.T) {
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
			// The SPA shell boots so the in-app React 404 renders.
			assert.Contains(t, string(body), `id="root"`)
		})
	}
}

// A directory must not be served (no listing); it falls through to the 404 shell.
func TestServeSPA_DirectoryServesShellAs404(t *testing.T) {
	res := serve(t, web.NewHandler(newTestFS()), http.MethodGet, "/assets")
	defer res.Body.Close()

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestServeSPA_HeadHasNoBody(t *testing.T) {
	t.Run("unknown path", func(t *testing.T) {
		res := serve(t, web.NewHandler(newTestFS()), http.MethodHead, "/notvalid")
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		assert.Empty(t, body)
	})
	t.Run("known route", func(t *testing.T) {
		res := serve(t, web.NewHandler(newTestFS()), http.MethodHead, "/")
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Empty(t, body)
	})
}

func TestServeSPA_NilFSNotBuilt(t *testing.T) {
	res := serve(t, web.NewHandler(nil), http.MethodGet, "/")
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Contains(t, string(body), "Frontend not built")
}
