package handlers

import (
	"net/http"

	"github.com/autobrr/brrewery/internal/buildinfo"
	"github.com/autobrr/brrewery/internal/httputil"
)

type VersionHandler struct{}

func NewVersionHandler() *VersionHandler {
	return &VersionHandler{}
}

func (h *VersionHandler) Version(w http.ResponseWriter, _ *http.Request) {
	data, err := buildinfo.JSON()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to read version")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
