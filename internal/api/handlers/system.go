package handlers

import (
	"errors"
	"net/http"

	"github.com/autobrr/brrewery/internal/httputil"
	"github.com/autobrr/brrewery/internal/system"
)

type SystemHandler struct {
	collector *system.Collector
}

func NewSystemHandler(collector *system.Collector) *SystemHandler {
	return &SystemHandler{collector: collector}
}

func (h *SystemHandler) Get(w http.ResponseWriter, _ *http.Request) {
	info, err := h.collector.Collect()
	if err != nil {
		if errors.Is(err, system.ErrUnsupported) {
			httputil.WriteError(w, http.StatusNotImplemented, "System metrics not available on this platform")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to collect system metrics")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, info)
}
