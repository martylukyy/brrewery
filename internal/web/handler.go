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

	if path != "index.html" {
		if file, err := h.fs.Open(path); err == nil {
			defer file.Close()
			stat, statErr := file.Stat()
			if statErr == nil && !stat.IsDir() {
				serveFile(w, r, path, file, stat)
				return
			}
		}
	}

	file, err := h.fs.Open("index.html")
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	serveFile(w, r, "index.html", file, stat)
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
