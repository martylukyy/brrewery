package detect

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/autobrr/brrewery/internal/apps/model"
)

const userPlaceholder = "{user}"

var standardBinaryDirs = []string{"/usr/local/bin", "/usr/bin", "/bin"}

// Evaluator checks filesystem and systemd state for app detection.
type Evaluator struct {
	lookPath         func(string) (string, error)
	systemctlActive  func(context.Context, string) error
	systemctlEnabled func(context.Context, string) error
	stat             func(string) (os.FileInfo, error)
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		lookPath: exec.LookPath,
		systemctlActive: func(ctx context.Context, unit string) error {
			return systemctlQuiet(ctx, "is-active", unit)
		},
		systemctlEnabled: func(ctx context.Context, unit string) error {
			return systemctlQuiet(ctx, "is-enabled", unit)
		},
		stat: os.Stat,
	}
}

func systemctlQuiet(ctx context.Context, op, unit string) error {
	cmd := exec.CommandContext(ctx, "systemctl", op, "--quiet", unit)
	return cmd.Run()
}

func (e *Evaluator) Installed(spec *model.DetectionSpec) bool {
	return e.InstalledForUser(spec, "")
}

func (e *Evaluator) InstalledForUser(spec *model.DetectionSpec, username string) bool {
	if spec == nil {
		return false
	}
	if !e.checkBinaries(spec.Binaries) {
		return false
	}
	if !e.checkPaths(spec.Paths) {
		return false
	}
	if len(spec.SystemdUnits) > 0 && !e.checkUnits(spec.SystemdUnits) {
		return false
	}
	if len(spec.SystemdUserUnits) > 0 {
		if username == "" {
			return false
		}
		if !e.checkUnitsEnabled(expandUserUnits(spec.SystemdUserUnits, username)) {
			return false
		}
	}
	return e.hasChecks(spec)
}

func (e *Evaluator) hasChecks(spec *model.DetectionSpec) bool {
	return len(spec.Binaries) > 0 ||
		len(spec.SystemdUnits) > 0 ||
		len(spec.SystemdUserUnits) > 0 ||
		len(spec.Paths) > 0
}

func expandUserUnits(templates []string, username string) []string {
	out := make([]string, 0, len(templates))
	for _, template := range templates {
		template = strings.TrimSpace(template)
		if template == "" {
			continue
		}
		out = append(out, strings.ReplaceAll(template, userPlaceholder, username))
	}
	return out
}

func (e *Evaluator) checkBinaries(binaries []string) bool {
	for _, b := range binaries {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		if !e.binaryPresent(b) {
			return false
		}
	}
	return true
}

func (e *Evaluator) binaryPresent(name string) bool {
	if _, err := e.lookPath(name); err == nil {
		return true
	}
	for _, dir := range standardBinaryDirs {
		if _, err := e.stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func (e *Evaluator) checkUnits(units []string) bool {
	ctx := context.Background()
	for _, unit := range units {
		unit = strings.TrimSpace(unit)
		if unit == "" {
			continue
		}
		if err := e.systemctlActive(ctx, unit); err != nil {
			return false
		}
	}
	return true
}

func (e *Evaluator) checkUnitsEnabled(units []string) bool {
	ctx := context.Background()
	for _, unit := range units {
		unit = strings.TrimSpace(unit)
		if unit == "" {
			continue
		}
		if err := e.systemctlEnabled(ctx, unit); err != nil {
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

func (e *Evaluator) DependenciesSatisfied(username string, deps []string, lookup func(string) model.DetectionSpec) bool {
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		spec := lookup(dep)
		if !e.InstalledForUser(&spec, username) {
			return false
		}
	}
	return true
}
