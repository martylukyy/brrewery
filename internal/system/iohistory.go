package system

import (
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	// SampleInterval is how often the daemon reads the I/O counters. Every
	// charted rate is derived from samples taken at this cadence.
	SampleInterval = time.Second
	// HistoryWindow is how much I/O history the daemon keeps in memory.
	HistoryWindow = 24 * time.Hour
	// SmoothingWindow is the trailing moving average applied to sampled rates
	// before they are charted, so a single-second spike does not dominate the
	// plot.
	SmoothingWindow = 5 * time.Second
	// MaxQueryPoints caps how many buckets one query may ask for. Charts ask
	// for at most one point per pixel of plot width, so this is far above any
	// real display.
	MaxQueryPoints = 4000

	historyCapacity = int(HistoryWindow / SampleInterval)
)

// Series keys used by the charts. The network series and the per-disk series
// carry the same shape, so the ring stores them as inbound/outbound and the
// query labels them per source.
const (
	SeriesRx    = "rx"
	SeriesTx    = "tx"
	SeriesRead  = "read"
	SeriesWrite = "write"
)

// IOCounters is one cumulative reading of the counters the throughput charts
// derive their rates from.
type IOCounters struct {
	Network NetworkCounters
	Disks   map[string]DiskIOCounters
}

// ratesRing holds the two rate series one throughput chart draws: receive and
// transmit for the network, read and write for a disk. Both are ring buffers
// aligned slot-for-slot with History.times; NaN marks a slot with no reading.
type ratesRing struct {
	inbound  []float64
	outbound []float64
	lastAt   int64
}

func newRatesRing(capacity int) *ratesRing {
	ring := &ratesRing{
		inbound:  make([]float64, capacity),
		outbound: make([]float64, capacity),
	}
	for i := range ring.inbound {
		ring.inbound[i] = math.NaN()
		ring.outbound[i] = math.NaN()
	}
	return ring
}

func (r *ratesRing) clear(slot int) {
	r.inbound[slot] = math.NaN()
	r.outbound[slot] = math.NaN()
}

func (r *ratesRing) set(slot int, inbound, outbound float64, at int64) {
	r.inbound[slot] = inbound
	r.outbound[slot] = outbound
	r.lastAt = at
}

// History is the daemon's in-memory record of recent throughput. It keeps
// HistoryWindow worth of per-second rates for the network and for every
// monitored mount, so a dashboard that just opened can draw the full window
// instead of building history from scratch in the browser.
type History struct {
	mu       sync.Mutex
	capacity int
	times    []int64 // sample time (unix millis) per slot
	head     int     // slot the next sample is written to
	filled   int     // slots holding a sample, up to capacity

	network *ratesRing
	disks   map[string]*ratesRing

	prev   IOCounters
	prevAt time.Time
	seeded bool
}

func NewHistory() *History {
	return newHistory(historyCapacity)
}

func newHistory(capacity int) *History {
	return &History{
		capacity: capacity,
		times:    make([]int64, capacity),
		network:  newRatesRing(capacity),
		disks:    map[string]*ratesRing{},
	}
}

// Record derives per-second rates from the previous reading and stores them.
// The first call only seeds the baseline. History takes ownership of the
// Disks map in counters.
func (h *History) Record(at time.Time, counters IOCounters) {
	h.mu.Lock()
	defer h.mu.Unlock()

	prev, prevAt, seeded := h.prev, h.prevAt, h.seeded
	h.prev, h.prevAt, h.seeded = counters, at, true

	seconds := at.Sub(prevAt).Seconds()
	if !seeded || seconds <= 0 {
		return
	}

	slot := h.advance(at)
	h.network.set(
		slot,
		ratePerSecond(counters.Network.RxBytes, prev.Network.RxBytes, seconds),
		ratePerSecond(counters.Network.TxBytes, prev.Network.TxBytes, seconds),
		at.UnixMilli(),
	)

	for mount, current := range counters.Disks {
		before, ok := prev.Disks[mount]
		if !ok {
			// First sighting of this mount: no baseline to subtract yet.
			continue
		}
		h.diskRing(mount).set(
			slot,
			ratePerSecond(current.ReadBytes, before.ReadBytes, seconds),
			ratePerSecond(current.WriteBytes, before.WriteBytes, seconds),
			at.UnixMilli(),
		)
	}
}

// advance claims the next ring slot and blanks it in every series, so a series
// that has no reading this tick leaves a gap rather than showing the value the
// slot held a full window ago.
func (h *History) advance(at time.Time) int {
	slot := h.head
	h.times[slot] = at.UnixMilli()
	h.head = (h.head + 1) % h.capacity
	if h.filled < h.capacity {
		h.filled++
	}

	h.network.clear(slot)
	for mount, ring := range h.disks {
		if at.UnixMilli()-ring.lastAt > HistoryWindow.Milliseconds() {
			// Nothing left in the window; drop the mount rather than keep its
			// buffers around for a disk that is gone.
			delete(h.disks, mount)
			continue
		}
		ring.clear(slot)
	}
	return slot
}

func (h *History) diskRing(mount string) *ratesRing {
	ring, ok := h.disks[mount]
	if !ok {
		ring = newRatesRing(h.capacity)
		h.disks[mount] = ring
	}
	return ring
}

// ratePerSecond converts a counter delta into a per-second rate. A counter that
// went backwards means the device was replaced or the counter wrapped; report
// no throughput rather than a bogus spike.
func ratePerSecond(current, previous uint64, seconds float64) float64 {
	if seconds <= 0 || current < previous {
		return 0
	}
	return float64(current-previous) / seconds
}

// HistoryQuery selects the series and resolution a chart wants to draw.
type HistoryQuery struct {
	// Window is how far back from now to plot.
	Window time.Duration
	// Points is how many buckets to return; the caller sizes this to the chart
	// width so no more points are sent than can be drawn.
	Points int
	// Mount selects a disk series; empty selects the network series.
	Mount string
}

// RatePoints is one downsampled chart series in bytes per second. Buckets with
// no sample marshal as null so the chart can leave a gap instead of drawing a
// zero it never measured.
type RatePoints []float64

func (p RatePoints) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, len(p)*8+2)
	buf = append(buf, '[')
	for i, value := range p {
		if i > 0 {
			buf = append(buf, ',')
		}
		if math.IsNaN(value) {
			buf = append(buf, "null"...)
			continue
		}
		// Whole bytes per second: the extra precision would only bloat the
		// response, which is sent once per second per chart.
		buf = strconv.AppendInt(buf, int64(math.Round(value)), 10)
	}
	return append(buf, ']'), nil
}

// ThroughputSeries is one named line of a throughput chart.
type ThroughputSeries struct {
	Key    string     `json:"key"`
	Points RatePoints `json:"points"`
}

// IOHistoryReport is the downsampled answer to a HistoryQuery. Buckets are
// evenly spaced: bucket i starts at StartMs + i*BucketSeconds.
type IOHistoryReport struct {
	StartMs          int64              `json:"start_ms"`
	EndMs            int64              `json:"end_ms"`
	BucketSeconds    float64            `json:"bucket_seconds"`
	SampleSeconds    float64            `json:"sample_seconds"`
	SmoothingSeconds float64            `json:"smoothing_seconds"`
	Series           []ThroughputSeries `json:"series"`
}

// bucketWindow is the resolved time grid a query is downsampled onto.
type bucketWindow struct {
	startMs  int64
	endMs    int64
	bucketMs int64
	points   int
}

// Query returns the requested window smoothed by a trailing moving average and
// downsampled to Points buckets. An unknown mount yields an all-gap series
// rather than an error: it simply has no samples in the window yet.
func (h *History) Query(now time.Time, query HistoryQuery) IOHistoryReport {
	window := resolveBucketWindow(now, query)

	h.mu.Lock()
	defer h.mu.Unlock()

	ring, inKey, outKey := h.network, SeriesRx, SeriesTx
	if query.Mount != "" {
		ring, inKey, outKey = h.disks[query.Mount], SeriesRead, SeriesWrite
	}

	var inbound, outbound []float64
	if ring != nil {
		inbound, outbound = ring.inbound, ring.outbound
	}

	return IOHistoryReport{
		StartMs:          window.startMs,
		EndMs:            window.endMs,
		BucketSeconds:    float64(window.bucketMs) / 1000,
		SampleSeconds:    SampleInterval.Seconds(),
		SmoothingSeconds: SmoothingWindow.Seconds(),
		Series: []ThroughputSeries{
			{Key: inKey, Points: h.downsample(inbound, window)},
			{Key: outKey, Points: h.downsample(outbound, window)},
		},
	}
}

func resolveBucketWindow(now time.Time, query HistoryQuery) bucketWindow {
	points := min(max(query.Points, 1), MaxQueryPoints)
	windowMs := query.Window.Milliseconds()
	// One bucket can never be finer than the sample rate; asking for more
	// points than there are samples just yields empty buckets.
	points = min(points, int(windowMs/SampleInterval.Milliseconds()))
	points = max(points, 1)
	bucketMs := max(windowMs/int64(points), 1)

	// Anchor the grid to absolute time rather than to now. Buckets tied to the
	// query time slide by the poll interval, which does not divide the bucket
	// width once the width comes from the chart's pixels rather than the sample
	// rate; every sample would then regroup with different neighbours a second
	// later and the whole plotted history would shimmer. Snapped to absolute
	// time a sample always lands in the same bucket, so only the newest, still
	// filling bucket changes between requests.
	endMs := ceilTo(now.UnixMilli(), bucketMs)
	return bucketWindow{
		startMs:  endMs - int64(points)*bucketMs,
		endMs:    endMs,
		bucketMs: bucketMs,
		points:   points,
	}
}

// ceilTo rounds ms up to the next multiple of step.
func ceilTo(ms, step int64) int64 {
	return (ms + step - 1) / step * step
}

// downsample walks the retained samples once, smoothing each with a trailing
// moving average and averaging the result into its bucket. Averaging (rather
// than picking a sample per bucket) keeps the plotted area proportional to the
// bytes actually transferred at every zoom level.
func (h *History) downsample(values []float64, window bucketWindow) RatePoints {
	points := make(RatePoints, window.points)
	sums := make([]float64, window.points)
	counts := make([]int, window.points)
	for i := range points {
		points[i] = math.NaN()
	}
	if values == nil {
		return points
	}

	smoothed := trailingMean{windowMs: SmoothingWindow.Milliseconds()}
	// Start one smoothing window early so the first plotted bucket sees the
	// same trailing average it would in a wider query.
	for k := h.firstSampleAtOrAfter(window.startMs - smoothed.windowMs); k < h.filled; k++ {
		slot := h.slot(k)
		at := h.times[slot]
		smoothed.add(at, values[slot])

		if at < window.startMs || at > window.endMs {
			continue
		}
		value := smoothed.value()
		if math.IsNaN(value) {
			continue
		}
		bucket := min(int((at-window.startMs)/window.bucketMs), window.points-1)
		sums[bucket] += value
		counts[bucket]++
	}

	// Sample times drift against the bucket grid, so at the finest zoom a bucket
	// can fall between two samples — including the partial bucket at the leading
	// edge. A hole exactly one bucket wide is that artifact rather than missing
	// data, so carry the previous value across it and leave longer holes as the
	// real gaps they are.
	for i := range points {
		switch {
		case counts[i] > 0:
			points[i] = sums[i] / float64(counts[i])
		case i > 0 && counts[i-1] > 0:
			points[i] = points[i-1]
		}
	}
	return points
}

// slot maps a chronological position (0 = oldest retained sample) to its ring
// slot.
func (h *History) slot(k int) int {
	return ((h.head-h.filled+k)%h.capacity + h.capacity) % h.capacity
}

// firstSampleAtOrAfter returns the chronological position of the oldest
// retained sample at or after ms. Sample times increase monotonically, so a
// binary search keeps short windows cheap even with a full 24h buffer.
func (h *History) firstSampleAtOrAfter(ms int64) int {
	return sort.Search(h.filled, func(k int) bool {
		return h.times[h.slot(k)] >= ms
	})
}

// trailingMean is the moving average of the values recorded within a trailing
// time window. It holds only the handful of samples inside that window, so the
// mean is recomputed exactly instead of drifting a running sum.
type trailingMean struct {
	windowMs int64
	times    []int64
	values   []float64
}

func (t *trailingMean) add(at int64, value float64) {
	t.evictBefore(at)
	if math.IsNaN(value) {
		return
	}
	t.times = append(t.times, at)
	t.values = append(t.values, value)
}

func (t *trailingMean) evictBefore(now int64) {
	drop := 0
	for drop < len(t.times) && now-t.times[drop] >= t.windowMs {
		drop++
	}
	if drop == 0 {
		return
	}
	t.times = append(t.times[:0], t.times[drop:]...)
	t.values = append(t.values[:0], t.values[drop:]...)
}

func (t *trailingMean) value() float64 {
	if len(t.values) == 0 {
		return math.NaN()
	}
	var sum float64
	for _, value := range t.values {
		sum += value
	}
	return sum / float64(len(t.values))
}
