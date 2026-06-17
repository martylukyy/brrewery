package detect

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/brrewery/internal/apps/model"
)

func TestInstalled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec model.DetectionSpec
		eval *Evaluator
		want bool
	}{
		{
			name: "binary found",
			spec: model.DetectionSpec{Binaries: []string{"nginx"}},
			eval: &Evaluator{
				lookPath: func(string) (string, error) { return "/usr/sbin/nginx", nil },
			},
			want: true,
		},
		{
			name: "binary missing",
			spec: model.DetectionSpec{Binaries: []string{"missing"}},
			eval: &Evaluator{
				lookPath: func(string) (string, error) { return "", errors.New("not found") },
				stat:     func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
			},
			want: false,
		},
		{
			name: "unit file present",
			spec: model.DetectionSpec{SystemdUnits: []string{"nginx.service"}},
			eval: &Evaluator{
				systemctlPresent: func(context.Context, string) error { return nil },
			},
			want: true,
		},
		{
			name: "unit file missing",
			spec: model.DetectionSpec{SystemdUnits: []string{"nginx.service"}},
			eval: &Evaluator{
				systemctlPresent: func(context.Context, string) error { return errors.New("no such unit") },
			},
			want: false,
		},
		{
			name: "stopped unit still installed",
			spec: model.DetectionSpec{SystemdUnits: []string{"deluged.service"}},
			eval: &Evaluator{
				// Service is down, but the unit file is installed: the app stays
				// detected so its toggle remains reachable.
				systemctlPresent: func(context.Context, string) error { return nil },
				systemctlActive:  func(context.Context, string) error { return errors.New("inactive") },
				systemctlEnabled: func(context.Context, string) error { return errors.New("disabled") },
			},
			want: true,
		},
		{
			name: "path exists",
			spec: model.DetectionSpec{Paths: []string{"/etc/nginx"}},
			eval: &Evaluator{
				stat: func(string) (os.FileInfo, error) {
					return fakeFileInfo{}, nil
				},
			},
			want: true,
		},
		{
			name: "no checks",
			spec: model.DetectionSpec{},
			eval: NewEvaluator(),
			want: false,
		},
		{
			name: "user scoped unit present",
			spec: model.DetectionSpec{
				Binaries:         []string{"autobrr"},
				SystemdUserUnits: []string{"autobrr@{user}.service"},
			},
			eval: &Evaluator{
				lookPath: func(string) (string, error) { return "/usr/local/bin/autobrr", nil },
				systemctlPresent: func(_ context.Context, unit string) error {
					if unit == "autobrr@admin.service" {
						return nil
					}
					return errors.New("no such unit")
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.eval.Installed(&tt.spec)
			if tt.name == "user scoped unit present" {
				got = tt.eval.InstalledForUser(&tt.spec, "admin")
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBinaryPresent_fallsBackToStandardDirs(t *testing.T) {
	t.Parallel()

	eval := &Evaluator{
		lookPath: func(string) (string, error) { return "", errors.New("not in path") },
		stat: func(path string) (os.FileInfo, error) {
			if path == "/usr/local/bin/qbittorrent-nox" {
				return fakeFileInfo{}, nil
			}
			return nil, os.ErrNotExist
		},
	}

	assert.True(t, eval.binaryPresent("qbittorrent-nox"))
}

func TestInstalledForUserRequiresUsernameForUserUnits(t *testing.T) {
	t.Parallel()

	eval := &Evaluator{
		lookPath:         func(string) (string, error) { return "/usr/local/bin/autobrr", nil },
		systemctlPresent: func(context.Context, string) error { return nil },
	}
	spec := model.DetectionSpec{
		Binaries:         []string{"autobrr"},
		SystemdUserUnits: []string{"autobrr@{user}.service"},
	}

	assert.False(t, eval.InstalledForUser(&spec, ""))
	assert.True(t, eval.InstalledForUser(&spec, "admin"))
}

func TestServiceStatus(t *testing.T) {
	t.Parallel()

	t.Run("no units", func(t *testing.T) {
		t.Parallel()
		_, ok := NewEvaluator().ServiceStatus(&model.DetectionSpec{Binaries: []string{"deluged"}}, "admin")
		assert.False(t, ok)
	})

	t.Run("user units without username", func(t *testing.T) {
		t.Parallel()
		spec := &model.DetectionSpec{SystemdUserUnits: []string{"autobrr@{user}.service"}}
		_, ok := NewEvaluator().ServiceStatus(spec, "")
		assert.False(t, ok)
	})

	t.Run("running and enabled", func(t *testing.T) {
		t.Parallel()
		eval := &Evaluator{
			systemctlActive:  func(context.Context, string) error { return nil },
			systemctlEnabled: func(context.Context, string) error { return nil },
			systemctlFailing: func(context.Context, string) bool { return false },
		}
		got, ok := eval.ServiceStatus(&model.DetectionSpec{SystemdUnits: []string{"deluged.service"}}, "")
		assert.True(t, ok)
		assert.Equal(t, []string{"deluged.service"}, got.Units)
		assert.True(t, got.Active)
		assert.True(t, got.Enabled)
		assert.False(t, got.Failing)
	})

	t.Run("stopped and disabled with expanded user unit", func(t *testing.T) {
		t.Parallel()
		eval := &Evaluator{
			systemctlActive:  func(context.Context, string) error { return errors.New("inactive") },
			systemctlEnabled: func(context.Context, string) error { return errors.New("disabled") },
			systemctlFailing: func(context.Context, string) bool { return false },
		}
		got, ok := eval.ServiceStatus(&model.DetectionSpec{SystemdUserUnits: []string{"sonarr@{user}.service"}}, "admin")
		assert.True(t, ok)
		assert.Equal(t, []string{"sonarr@admin.service"}, got.Units)
		assert.False(t, got.Active)
		assert.False(t, got.Enabled)
		assert.False(t, got.Failing)
	})

	t.Run("crash-looping unit reports failing while inactive", func(t *testing.T) {
		t.Parallel()
		// A crash-looping unit never reaches "running" (is-active fails) but is
		// still enabled; Failing must be reported alongside Active=false so the
		// dashboard can draw its red backdrop.
		eval := &Evaluator{
			systemctlActive:  func(context.Context, string) error { return errors.New("activating") },
			systemctlEnabled: func(context.Context, string) error { return nil },
			systemctlFailing: func(_ context.Context, unit string) bool { return unit == "deluge-web@admin.service" },
		}
		spec := &model.DetectionSpec{
			SystemdUserUnits: []string{"deluged@{user}.service", "deluge-web@{user}.service"},
		}
		got, ok := eval.ServiceStatus(spec, "admin")
		assert.True(t, ok)
		assert.Equal(t, []string{"deluged@admin.service", "deluge-web@admin.service"}, got.Units)
		assert.False(t, got.Active)
		assert.True(t, got.Enabled)
		assert.True(t, got.Failing)
	})
}

func TestParseShowState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		out        string
		wantActive string
		wantSub    string
	}{
		{
			name:       "auto-restart loop",
			out:        "ActiveState=activating\nSubState=auto-restart\n",
			wantActive: "activating",
			wantSub:    "auto-restart",
		},
		{
			name:       "failed",
			out:        "ActiveState=failed\nSubState=failed\n",
			wantActive: "failed",
			wantSub:    "failed",
		},
		{
			name:       "running",
			out:        "ActiveState=active\nSubState=running\n",
			wantActive: "active",
			wantSub:    "running",
		},
		{
			name:       "ignores unrelated properties",
			out:        "Id=deluge-web@admin.service\nActiveState=active\nResult=success\nSubState=running",
			wantActive: "active",
			wantSub:    "running",
		},
		{
			name: "empty output",
			out:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			active, sub := parseShowState([]byte(tt.out))
			assert.Equal(t, tt.wantActive, active)
			assert.Equal(t, tt.wantSub, sub)
		})
	}
}

type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "nginx" }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return 0o755 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return true }
func (fakeFileInfo) Sys() any           { return nil }
