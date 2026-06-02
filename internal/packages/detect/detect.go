package detect

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/autobrr/brrewery/internal/packages/model"
)

// Evaluator checks filesystem and systemd state for package detection.
type Evaluator struct {
	lookPath  func(string) (string, error)
	systemctl func(context.Context, string) error
	stat      func(string) (os.FileInfo, error)
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		lookPath: exec.LookPath,
		systemctl: func(ctx context.Context, unit string) error {
			cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", unit)
			return cmd.Run()
		},
		stat: os.Stat,
	}
}

func (e *Evaluator) Installed(spec *model.DetectionSpec) bool {
	if spec == nil {
		return false
	}
	if !e.checkBinaries(spec.Binaries) {
		return false
	}
	if !e.checkUnits(spec.SystemdUnits) {
		return false
	}
	if !e.checkPaths(spec.Paths) {
		return false
	}
	return e.hasChecks(spec)
}

func (e *Evaluator) hasChecks(spec *model.DetectionSpec) bool {
	return len(spec.Binaries) > 0 || len(spec.SystemdUnits) > 0 || len(spec.Paths) > 0
}

func (e *Evaluator) checkBinaries(binaries []string) bool {
	for _, b := range binaries {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		if _, err := e.lookPath(b); err != nil {
			return false
		}
	}
	return true
}

func (e *Evaluator) checkUnits(units []string) bool {
	ctx := context.Background()
	for _, unit := range units {
		unit = strings.TrimSpace(unit)
		if unit == "" {
			continue
		}
		if err := e.systemctl(ctx, unit); err != nil {
			return false
		}
	}
	return true
}

func (e *Evaluator) checkPaths(paths []string) bool {
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, err := e.stat(p); err != nil {
			return false
		}
	}
	return true
}

func (e *Evaluator) DependenciesSatisfied(deps []string, lookup func(string) model.DetectionSpec) bool {
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		spec := lookup(dep)
		if !e.Installed(&spec) {
			return false
		}
	}
	return true
}
