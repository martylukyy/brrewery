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
// 200). They mirror the TanStack route tree in web/src/router.tsx and the nginx
// vhost allowlist. Every other non-asset path is served the same index.html
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

// serveShell writes index.html with an explicit status. A 404 is marked
// no-store so a not-found response is never cached as a healthy page; HEAD gets
// the headers and status without a body.
func (h *Handler) serveShell(w http.ResponseWriter, r *http.Request, status int) {
	file, err := h.fs.Open("index.html")
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
