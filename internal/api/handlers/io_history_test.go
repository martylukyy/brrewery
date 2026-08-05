package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autobrr/brrewery/internal/api/handlers"
	"github.com/autobrr/brrewery/internal/system"
)

func ioHistoryResponse(t *testing.T, query string) *httptest.ResponseRecorder {
	t.Helper()

	history := system.NewHistory()
	now := time.Now()
	history.Record(now.Add(-2*time.Second), system.IOCounters{
		Network: system.NetworkCounters{RxBytes: 1000, TxBytes: 500},
		Disks:   map[string]system.DiskIOCounters{"/": {ReadBytes: 2000, WriteBytes: 1000}},
	})
	history.Record(now.Add(-time.Second), system.IOCounters{
		Network: system.NetworkCounters{RxBytes: 3000, TxBytes: 1500},
		Disks:   map[string]system.DiskIOCounters{"/": {ReadBytes: 6000, WriteBytes: 3000}},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/system/io-history?"+query, http.NoBody)
	handlers.NewIOHistoryHandler(history).Get(rec, req)
	return rec
}

func TestIOHistoryHandlerReturnsRequestedPointCount(t *testing.T) {
	rec := ioHistoryResponse(t, "range=1m&points=12&mount=/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var report struct {
		BucketSeconds    float64 `json:"bucket_seconds"`
		SampleSeconds    float64 `json:"sample_seconds"`
		SmoothingSeconds float64 `json:"smoothing_seconds"`
		Series           []struct {
			Key    string     `json:"key"`
			Points []*float64 `json:"points"`
		} `json:"series"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if report.SampleSeconds != 1 || report.SmoothingSeconds != 5 || report.BucketSeconds != 5 {
		t.Fatalf("unexpected resolution: %+v", report)
	}
	if len(report.Series) != 2 || report.Series[0].Key != "read" || report.Series[1].Key != "write" {
		t.Fatalf("unexpected series: %+v", report.Series)
	}
	for _, series := range report.Series {
		if len(series.Points) != 12 {
			t.Fatalf("series %q returned %d points, want 12", series.Key, len(series.Points))
		}
	}

	// The one sampled second carries a 4000 B/s read rate; earlier buckets are gaps.
	read := report.Series[0].Points
	if last := read[len(read)-1]; last == nil || *last != 4000 {
		t.Fatalf("latest read bucket = %v, want 4000", last)
	}
	if read[0] != nil {
		t.Fatalf("bucket before the first sample = %v, want null", *read[0])
	}
}

func TestIOHistoryHandlerRejectsBadParameters(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "missing range", query: "points=10"},
		{name: "unparsable range", query: "range=forever&points=10"},
		{name: "range beyond retention", query: "range=48h&points=10"},
		{name: "missing points", query: "range=5m"},
		{name: "zero points", query: "range=5m&points=0"},
		{name: "more points than allowed", query: "range=5m&points=99999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := ioHistoryResponse(t, tt.query); rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}
