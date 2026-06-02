package handlers

import (
	"net/http"

	"github.com/autobrr/brrewery/internal/httputil"
	"github.com/autobrr/brrewery/internal/vnstat"
)

type VnstatHandler struct {
	collector *vnstat.Collector
}

func NewVnstatHandler(collector *vnstat.Collector) *VnstatHandler {
	return &VnstatHandler{collector: collector}
}

func (h *VnstatHandler) Get(w http.ResponseWriter, r *http.Request) {
	report, err := h.collector.Collect(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to read vnstat data")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, report)
}
