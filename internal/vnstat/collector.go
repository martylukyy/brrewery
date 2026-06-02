package vnstat

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const (
	dayLimit   = 30
	monthLimit = 12
	runTimeout = 10 * time.Second
)

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

func (c *Collector) Collect(ctx context.Context) (Report, error) {
	if _, err := c.runner.LookPath("vnstat"); err != nil {
		return Report{
			Installed: false,
			Message:   "vnstat is not installed on this system.",
		}, nil
	}

	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	dayJSON, err := c.runner.Output(runCtx, "vnstat", "--json", "d", fmt.Sprintf("%d", dayLimit))
	if err != nil {
		return Report{}, fmt.Errorf("vnstat days: %w", err)
	}

	monthJSON, err := c.runner.Output(runCtx, "vnstat", "--json", "m", fmt.Sprintf("%d", monthLimit))
	if err != nil {
		return Report{}, fmt.Errorf("vnstat months: %w", err)
	}

	return parseReport(dayJSON, monthJSON)
}
