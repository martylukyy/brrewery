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

func (h *Handler) ServeSPA(w http.ResponseWriter, r *http.Request) {
	if h.fs == nil {
		http.Error(w, "Frontend not built. Run 'make frontend'.", http.StatusNotFound)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// The frontend is a single page with no client-side router: the only valid
	// document is index.html ("/"). Any other path must resolve to a real static
	// asset, otherwise it is genuinely not found. Falling back to index.html for
	// unknown paths (the usual SPA trick) would answer /notvalid/ with a 200 and
	// the dashboard, hiding broken links and bad bookmarks behind a healthy page.
	if file, err := h.fs.Open(path); err == nil {
		defer file.Close()
		if stat, statErr := file.Stat(); statErr == nil && !stat.IsDir() {
			serveFile(w, r, path, file, stat)
			return
		}
	}

	h.serveNotFound(w, r)
}

// serveNotFound writes a 404 backed by the embedded 404.html error page. That
// page is the single source of truth shared with the production nginx vhost
// (error_page 404 /404.html), so the standalone Go server and the nginx-fronted
// deployment render the same page. A tiny inline fallback covers the case where
// the frontend has not been built and 404.html is absent from the bundle.
func (h *Handler) serveNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)
	if r.Method == http.MethodHead {
		return
	}

	// Prefer the bundled error page; the inline fallback is only for a bundle
	// that predates 404.html. Once the file opens we commit to it — a mid-write
	// copy error means the client went away, not that the fallback should be
	// appended to a half-written page.
	if h.fs != nil {
		if file, err := h.fs.Open("404.html"); err == nil {
			defer file.Close()
			_, _ = io.Copy(w, file)
			return
		}
	}
	_, _ = io.WriteString(w, fallbackNotFound)
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

const fallbackNotFound = `<!DOCTYPE html>
<html lang="en">
  <head><meta charset="UTF-8" /><title>404 — brrewery</title></head>
  <body><h1>404 — Page not found</h1></body>
</html>
`
