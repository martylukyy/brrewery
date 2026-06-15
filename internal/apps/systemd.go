package apps

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// serviceController applies the on/off state of an app's systemd units.
// brrewery runs as root (see contrib/systemd/brrewery.service), so it invokes
// systemctl directly rather than escalating per call.
type serviceController interface {
	// SetEnabled enables and starts (on) or disables and stops (off) the units
	// as a single transition.
	SetEnabled(ctx context.Context, units []string, on bool) error
}

type systemctlController struct{}

func (systemctlController) SetEnabled(ctx context.Context, units []string, on bool) error {
	if len(units) == 0 {
		return nil
	}
	// enable/disable --now flips the enablement and the running state together,
	// matching the dashboard toggle's "start & enable" / "stop & disable".
	op := "disable"
	if on {
		op = "enable"
	}
	args := append([]string{op, "--now"}, units...)
	out, err := exec.CommandContext(ctx, "systemctl", args...).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("systemctl %s: %s", op, msg)
		}
		return fmt.Errorf("systemctl %s: %w", op, err)
	}
	return nil
}
