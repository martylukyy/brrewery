package handlers

import (
	"net/http"
	"strconv"

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
	days, err := positiveQueryParam(r, "days")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid 'days' parameter")
		return
	}
	months, err := positiveQueryParam(r, "months")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid 'months' parameter")
		return
	}

	report, err := h.collector.Collect(r.Context(), days, months)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to read vnstat data")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, report)
}

// positiveQueryParam parses a required query parameter that must be a positive
// integer, returning an error when it is missing, malformed, or non-positive.
func positiveQueryParam(r *http.Request, name string) (int, error) {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || value <= 0 {
		return 0, strconv.ErrSyntax
	}
	return value, nil
}
