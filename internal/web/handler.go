package web

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

type Handler struct {
	fs http.FileSystem
}

func NewHandler(fsys fs.FS) *Handler {
	if fsys == nil {
		return &Handler{}
	}
	return &Handler{fs: http.FS(fsys)}
}

// knownRoutes are the client-side routes the SPA renders as a real page (HTTP
// 200). They mirror the TanStack route tree in web/src/router.tsx — add an entry
// per new route. This is the only such allowlist: the nginx vhost proxies every
// path here instead of resolving routes itself. Every other non-asset path is
// served the same index.html
// shell but with a 404 status, so the in-app <NotFound/> page renders against a
// genuine 404 instead of the dashboard masking a broken link with a 200.
var knownRoutes = map[string]bool{
	"":           true, // "/"
	"index.html": true,
	"login":      true,
}

func (h *Handler) ServeSPA(w http.ResponseWriter, r *http.Request) {
	if h.fs == nil {
		http.Error(w, "Frontend not built. Run 'make frontend'.", http.StatusNotFound)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")

	// A real static asset (js/css/image/…) is served as-is with a 200.
	if path != "" && path != "index.html" {
		if file, err := h.fs.Open(path); err == nil {
			defer file.Close()
			if stat, statErr := file.Stat(); statErr == nil && !stat.IsDir() {
				serveFile(w, r, path, file, stat)
				return
			}
		}
	}

	// Otherwise serve the SPA shell: a 200 for a known route, a true 404 for
	// anything else (the client router then renders the matching page).
	status := http.StatusNotFound
	if knownRoutes[path] {
		status = http.StatusOK
	}
	h.serveShell(w, r, status)
}

// serveShell writes index.html with an explicit status. The shell is always
// revalidated — its URL is stable across releases, so a cached copy would keep
// pointing at the previous build's hashed assets after an update. A 404 goes
// further and is never stored, so a not-found response is not kept around as a
// healthy page; HEAD gets the headers and status without a body.
func (h *Handler) serveShell(w http.ResponseWriter, r *http.Request, status int) {
	file, err := h.fs.Open("index.html")
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if status == http.StatusNotFound {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(w, file)
}

func serveFile(w http.ResponseWriter, r *http.Request, path string, file fs.File, stat fs.FileInfo) {
	ext := filepath.Ext(path)
	if ct := mime.TypeByExtension(ext); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", cacheControlFor(path))

	if rs, ok := file.(io.ReadSeeker); ok {
		http.ServeContent(w, r, path, stat.ModTime(), rs)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

// cacheControlFor picks the caching policy for a bundled asset. Files under
// assets/ carry Vite's content hash in their name, so a changed file is a
// changed URL and the response can be cached forever. Everything else (favicons,
// logos under public/, …) keeps a stable URL across releases and must be
// revalidated, or an update would leave a browser on the old copy.
//
// The header is explicit because the bundle is served from an embed.FS, whose
// entries all report a zero modtime: net/http then omits Last-Modified and no
// conditional request is possible.
func cacheControlFor(path string) string {
	if strings.HasPrefix(path, "assets/") {
		return "public, max-age=31536000, immutable"
	}
	return "no-cache"
}
