package system

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

var base = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// recordSeconds feeds one sample per second, taking the per-second network and
// disk rates for second i from rates[i]. Counters are cumulative, so the first
// entry only seeds the baseline.
func recordSeconds(h *History, rates []float64) {
	var read, rx uint64
	for i, rate := range rates {
		read += uint64(rate)
		rx += uint64(rate)
		h.Record(base.Add(time.Duration(i)*time.Second), IOCounters{
			Network: NetworkCounters{RxBytes: rx, TxBytes: rx / 2},
			Disks:   map[string]DiskIOCounters{"/": {ReadBytes: read, WriteBytes: read / 2}},
		})
	}
}

func queryAt(h *History, seconds int, query HistoryQuery) IOHistoryReport {
	return h.Query(base.Add(time.Duration(seconds)*time.Second), query)
}

func seriesPoints(t *testing.T, report IOHistoryReport, key string) RatePoints {
	t.Helper()
	for _, series := range report.Series {
		if series.Key == key {
			return series.Points
		}
	}
	t.Fatalf("series %q missing from report", key)
	return nil
}

func TestHistoryRecordNeedsBaseline(t *testing.T) {
	h := newHistory(16)
	h.Record(base, IOCounters{Network: NetworkCounters{RxBytes: 100}})

	report := queryAt(h, 1, HistoryQuery{Window: 10 * time.Second, Points: 10})
	for _, value := range seriesPoints(t, report, SeriesRx) {
		if !math.IsNaN(value) {
			t.Fatalf("expected no points from a single sample, got %v", value)
		}
	}
}

func TestHistorySmoothsWithTrailingMean(t *testing.T) {
	h := newHistory(64)
	// A single 1000 B/s spike at second 5, idle before and after.
	rates := make([]float64, 15)
	rates[5] = 1000
	recordSeconds(h, rates)

	// Window covers seconds 4-14 at one point per sample, so each point is the
	// trailing mean alone.
	report := queryAt(h, 14, HistoryQuery{Window: 10 * time.Second, Points: 10, Mount: "/"})
	points := seriesPoints(t, report, SeriesRead)

	// The 5s trailing mean spreads the spike evenly over the five samples that
	// see it, and it is gone from the sixth onwards.
	want := RatePoints{0, 200, 200, 200, 200, 200, 0, 0, 0, 0}
	for i, expected := range want {
		if points[i] != expected {
			t.Fatalf("point %d = %v, want %v (full series %v)", i, points[i], expected, points)
		}
	}
}

func TestHistoryBridgesSubBucketSamplingJitter(t *testing.T) {
	h := newHistory(64)
	// Samples drifting past the second boundary, as a ticker's jitter produces:
	// some 1s buckets get two samples and their neighbours get none.
	var read uint64
	for i := range 10 {
		read += 100
		at := base.Add(time.Duration(i)*time.Second + time.Duration(i*120)*time.Millisecond)
		h.Record(at, IOCounters{Disks: map[string]DiskIOCounters{"/": {ReadBytes: read}}})
	}

	report := queryAt(h, 10, HistoryQuery{Window: 10 * time.Second, Points: 10, Mount: "/"})
	points := seriesPoints(t, report, SeriesRead)

	// The first bucket predates the first rate, the rest must form an unbroken
	// line rather than alternating between values and gaps.
	for i, value := range points[1:] {
		if math.IsNaN(value) {
			t.Fatalf("bucket %d is a gap; jitter broke the line: %v", i+1, points)
		}
	}
}

func TestHistoryKeepsGapsWiderThanOneBucket(t *testing.T) {
	h := newHistory(64)
	recordSeconds(h, []float64{0, 100, 100})
	// Daemon stalls for four seconds, then resumes.
	h.Record(base.Add(7*time.Second), IOCounters{
		Disks: map[string]DiskIOCounters{"/": {ReadBytes: 900}},
	})

	report := queryAt(h, 7, HistoryQuery{Window: 7 * time.Second, Points: 7, Mount: "/"})
	points := seriesPoints(t, report, SeriesRead)

	// Buckets covering seconds 3-5 have no sample and no adjacent one to borrow.
	for _, i := range []int{4, 5} {
		if !math.IsNaN(points[i]) {
			t.Fatalf("bucket %d = %v, want a gap across the stall: %v", i, points[i], points)
		}
	}
}

func TestHistoryKeepsPastBucketsStableAsTheWindowSlides(t *testing.T) {
	h := newHistory(2048)
	// Rates that vary sample to sample, so regrouping samples into different
	// buckets would visibly change the averages.
	rates := make([]float64, 901)
	for i := range rates {
		rates[i] = float64((i % 7) * 1000)
	}
	recordSeconds(h, rates)

	// 15 minutes over 640 points gives 1406ms buckets — a width the one-second
	// poll interval does not divide evenly, which is what used to reshuffle the
	// history on every request.
	query := HistoryQuery{Window: 15 * time.Minute, Points: 640}
	first := queryAt(h, 900, query)
	firstPoints := seriesPoints(t, first, SeriesRx)

	for second := 901; second <= 905; second++ {
		later := queryAt(h, second, query)
		laterPoints := seriesPoints(t, later, SeriesRx)

		// Aligned grids differ only by whole buckets, so the same instant is
		// found by shifting the index.
		bucketMs := int64(later.BucketSeconds * 1000)
		if (later.StartMs-first.StartMs)%bucketMs != 0 {
			t.Fatalf("grid drifted by a partial bucket at second %d", second)
		}
		shift := (later.StartMs - first.StartMs) / bucketMs

		// The newest bucket of the earlier report was still filling, so compare
		// everything before it.
		for i := shift; i < int64(len(firstPoints))-1; i++ {
			before, after := firstPoints[i], laterPoints[i-shift]
			if math.IsNaN(before) && math.IsNaN(after) {
				continue
			}
			if before != after {
				t.Fatalf(
					"bucket at index %d changed after %ds: %v -> %v",
					i, second-900, before, after,
				)
			}
		}
	}
}

func TestHistoryDownsamplesToRequestedPoints(t *testing.T) {
	h := newHistory(128)
	rates := make([]float64, 61)
	for i := range rates {
		rates[i] = 100
	}
	recordSeconds(h, rates)

	report := queryAt(h, 60, HistoryQuery{Window: time.Minute, Points: 10})
	points := seriesPoints(t, report, SeriesRx)

	if len(points) != 10 {
		t.Fatalf("got %d points, want 10", len(points))
	}
	if report.BucketSeconds != 6 {
		t.Fatalf("bucket seconds = %v, want 6", report.BucketSeconds)
	}
	for i, value := range points {
		if value != 100 {
			t.Fatalf("point %d = %v, want a steady 100", i, value)
		}
	}
}

func TestHistoryNeverReturnsMorePointsThanSamples(t *testing.T) {
	h := newHistory(128)
	report := queryAt(h, 0, HistoryQuery{Window: 10 * time.Second, Points: 500})

	if got := len(seriesPoints(t, report, SeriesRx)); got != 10 {
		t.Fatalf("got %d points for a 10s window, want 10", got)
	}
}

func TestHistoryReportsGapsForUnknownMount(t *testing.T) {
	h := newHistory(16)
	recordSeconds(h, []float64{0, 100, 100})

	report := queryAt(h, 3, HistoryQuery{Window: 10 * time.Second, Points: 5, Mount: "/mnt/gone"})
	for _, series := range report.Series {
		for _, value := range series.Points {
			if !math.IsNaN(value) {
				t.Fatalf("series %q has data for an unknown mount: %v", series.Key, value)
			}
		}
	}
}

func TestHistoryDropsSamplesOutsideTheWindow(t *testing.T) {
	// Capacity of 4 keeps only the four most recent samples.
	h := newHistory(4)
	recordSeconds(h, []float64{0, 100, 100, 100, 100, 100, 100})

	report := queryAt(h, 6, HistoryQuery{Window: 6 * time.Second, Points: 6, Mount: "/"})
	points := seriesPoints(t, report, SeriesRead)

	if !math.IsNaN(points[0]) {
		t.Fatalf("expected the evicted oldest bucket to be a gap, got %v", points[0])
	}
	if got := points[len(points)-1]; got != 100 {
		t.Fatalf("latest bucket = %v, want 100", got)
	}
}

func TestHistoryIgnoresCounterResets(t *testing.T) {
	h := newHistory(16)
	h.Record(base, IOCounters{Network: NetworkCounters{RxBytes: 10_000}})
	h.Record(base.Add(time.Second), IOCounters{Network: NetworkCounters{RxBytes: 5}})

	report := queryAt(h, 1, HistoryQuery{Window: 5 * time.Second, Points: 5})
	for _, value := range seriesPoints(t, report, SeriesRx) {
		if !math.IsNaN(value) && value != 0 {
			t.Fatalf("counter reset charted as %v, want 0", value)
		}
	}
}

func TestRatePointsMarshalGapsAsNull(t *testing.T) {
	encoded, err := json.Marshal(RatePoints{math.NaN(), 1024.4, 2048.6})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(encoded) != "[null,1024,2049]" {
		t.Fatalf("got %s, want [null,1024,2049]", encoded)
	}
}
