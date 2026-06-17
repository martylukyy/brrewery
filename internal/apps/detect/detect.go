package detect

import (
	"context"
	"io"
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
	systemctlFailing func(context.Context, string) bool
	systemctlPresent func(context.Context, string) error
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
		systemctlFailing: systemctlUnitFailing,
		systemctlPresent: systemctlUnitPresent,
		stat:             os.Stat,
	}
}

func systemctlQuiet(ctx context.Context, op, unit string) error {
	cmd := exec.CommandContext(ctx, "systemctl", op, "--quiet", unit)
	return cmd.Run()
}

// systemctlUnitFailing reports whether a unit is unhealthy: either it has failed
// outright (ActiveState=failed) or it is stuck restarting (SubState=auto-restart),
// the latter being how a crash-looping service such as a misconfigured deluge-web
// presents before systemd gives up. `is-failed` is not enough on its own — it
// only matches the terminal "failed" state, not the auto-restart loop. A cleanly
// stopped unit (inactive) or a normally starting one (activating/start) is not
// failing. `systemctl show` exits 0 even for unknown units (reporting inactive),
// so a non-nil error here means the probe itself failed; treat that as not-failing.
func systemctlUnitFailing(ctx context.Context, unit string) bool {
	out, err := exec.CommandContext(ctx, "systemctl", "show", unit,
		"--property=ActiveState", "--property=SubState").Output()
	if err != nil {
		return false
	}
	activeState, subState := parseShowState(out)
	return activeState == "failed" || subState == "auto-restart"
}

// parseShowState pulls ActiveState and SubState out of `systemctl show` key=value
// output, ignoring any other properties.
func parseShowState(out []byte) (activeState, subState string) {
	for _, line := range strings.Split(string(out), "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "ActiveState":
			activeState = strings.TrimSpace(val)
		case "SubState":
			subState = strings.TrimSpace(val)
		}
	}
	return activeState, subState
}

// systemctlUnitPresent reports (via a nil error) whether a unit file is
// installed, regardless of whether it is currently active or enabled. This is
// the persistent-artifact signal for detection: `systemctl cat` resolves the
// unit (including the template behind an instance like sonarr@user.service) and
// exits non-zero only when no such unit file exists.
func systemctlUnitPresent(ctx context.Context, unit string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "cat", unit)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
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
	// Installed reflects persistent artifacts, so units are detected by the unit
	// file existing — not by being active/enabled. That keeps a stopped or
	// disabled app listed (and its service toggle reachable). The live run state
	// is reported separately by ServiceStatus.
	if len(spec.SystemdUnits) > 0 && !e.checkUnitsPresent(spec.SystemdUnits) {
		return false
	}
	if len(spec.SystemdUserUnits) > 0 {
		if username == "" {
			return false
		}
		if !e.checkUnitsPresent(expandUserUnits(spec.SystemdUserUnits, username)) {
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

func (e *Evaluator) checkUnitsPresent(units []string) bool {
	ctx := context.Background()
	for _, unit := range units {
		unit = strings.TrimSpace(unit)
		if unit == "" {
			continue
		}
		if err := e.systemctlPresent(ctx, unit); err != nil {
			return false
		}
	}
	return true
}

// ServiceStatus reports the live systemd state of the controllable units for an
// app, with {user} instance units expanded. ok is false when the spec declares
// no units (or declares user units without a username), meaning the app has no
// service to toggle. Active/Enabled are true only when every unit is; Failing is
// true when any unit is unhealthy (failed or crash-looping).
func (e *Evaluator) ServiceStatus(spec *model.DetectionSpec, username string) (model.ServiceStatus, bool) {
	if spec == nil {
		return model.ServiceStatus{}, false
	}

	units := make([]string, 0, len(spec.SystemdUnits)+len(spec.SystemdUserUnits))
	for _, unit := range spec.SystemdUnits {
		if unit = strings.TrimSpace(unit); unit != "" {
			units = append(units, unit)
		}
	}
	if len(spec.SystemdUserUnits) > 0 {
		if username == "" {
			return model.ServiceStatus{}, false
		}
		units = append(units, expandUserUnits(spec.SystemdUserUnits, username)...)
	}
	if len(units) == 0 {
		return model.ServiceStatus{}, false
	}

	ctx := context.Background()
	status := model.ServiceStatus{Units: units, Active: true, Enabled: true}
	for _, unit := range units {
		if e.systemctlActive(ctx, unit) != nil {
			status.Active = false
		}
		if e.systemctlEnabled(ctx, unit) != nil {
			status.Enabled = false
		}
		if e.systemctlFailing(ctx, unit) {
			status.Failing = true
		}
	}
	return status, true
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
