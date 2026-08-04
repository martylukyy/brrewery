package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/autobrr/brrewery/internal/httputil"
	"github.com/autobrr/brrewery/internal/system"
)

// IOHistoryHandler serves the throughput history the daemon keeps in memory,
// downsampled to the number of points the requesting chart can draw.
type IOHistoryHandler struct {
	history *system.History
}

func NewIOHistoryHandler(history *system.History) *IOHistoryHandler {
	return &IOHistoryHandler{history: history}
}

func (h *IOHistoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	window, ok := historyRangeParam(r)
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid 'range' parameter, expected a duration up to 24h")
		return
	}

	points, ok := historyPointsParam(r)
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "Invalid 'points' parameter")
		return
	}

	// An unknown mount is not an error: it simply has no samples in the window,
	// which the report expresses as gaps.
	report := h.history.Query(time.Now(), system.HistoryQuery{
		Window: window,
		Points: points,
		Mount:  r.URL.Query().Get("mount"),
	})
	httputil.WriteJSON(w, http.StatusOK, report)
}

// historyRangeParam parses the plotted time range, given as a Go duration
// ("5m", "24h") and bounded by how much history the daemon retains.
func historyRangeParam(r *http.Request) (time.Duration, bool) {
	window, err := time.ParseDuration(r.URL.Query().Get("range"))
	if err != nil || window < system.SampleInterval || window > system.HistoryWindow {
		return 0, false
	}
	return window, true
}

// historyPointsParam parses how many buckets the chart wants — normally its
// width in pixels, so the response never carries more points than can be drawn.
func historyPointsParam(r *http.Request) (int, bool) {
	points, err := strconv.Atoi(r.URL.Query().Get("points"))
	if err != nil || points < 1 || points > system.MaxQueryPoints {
		return 0, false
	}
	return points, true
}
