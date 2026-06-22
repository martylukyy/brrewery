package vnstat

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

const runTimeout = 10 * time.Second

var ErrNotInstalled = errors.New("vnstat is not installed")

type commandRunner interface {
	LookPath(name string) (string, error)
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (execRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

type Collector struct {
	runner commandRunner
}

func NewCollector() *Collector {
	return &Collector{runner: execRunner{}}
}

// Collect reads the last dayLimit days and monthLimit months of traffic. The
// limits are supplied by the caller so the available ranges can be defined by
// the frontend rather than hardcoded here.
func (c *Collector) Collect(ctx context.Context, dayLimit, monthLimit int) (Report, error) {
	if _, err := c.runner.LookPath("vnstat"); err != nil {
		return Report{
			Installed: false,
			Message:   "vnstat is not installed on this system.",
		}, nil
	}

	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	dayJSON, err := c.runner.Output(runCtx, "vnstat", "--json", "d", strconv.Itoa(dayLimit))
	if err != nil {
		return Report{}, fmt.Errorf("vnstat days: %w", err)
	}

	monthJSON, err := c.runner.Output(runCtx, "vnstat", "--json", "m", strconv.Itoa(monthLimit))
	if err != nil {
		return Report{}, fmt.Errorf("vnstat months: %w", err)
	}

	return parseReport(dayJSON, monthJSON)
}
