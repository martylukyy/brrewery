package system

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
)

// CounterSource reads the cumulative I/O counters a History samples.
type CounterSource interface {
	CollectIOCounters() (IOCounters, error)
}

// Run samples the counters on every tick until ctx is cancelled, so the
// dashboard's charts are backed by history the daemon collected all along
// rather than by whatever a browser tab happened to observe while it was open.
func (h *History) Run(ctx context.Context, source CounterSource, interval time.Duration, logger *zerolog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		counters, err := source.CollectIOCounters()
		switch {
		case errors.Is(err, ErrUnsupported):
			logger.Warn().Msg("I/O history unavailable on this platform")
			return
		case err != nil:
			// A counter read can fail transiently (a mount going away mid-read);
			// the missing sample shows up as a gap in the chart.
			logger.Debug().Err(err).Msg("io history sample failed")
		default:
			h.Record(time.Now(), counters)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
