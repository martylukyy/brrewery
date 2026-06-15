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
		}
		got, ok := eval.ServiceStatus(&model.DetectionSpec{SystemdUnits: []string{"deluged.service"}}, "")
		assert.True(t, ok)
		assert.Equal(t, []string{"deluged.service"}, got.Units)
		assert.True(t, got.Active)
		assert.True(t, got.Enabled)
	})

	t.Run("stopped and disabled with expanded user unit", func(t *testing.T) {
		t.Parallel()
		eval := &Evaluator{
			systemctlActive:  func(context.Context, string) error { return errors.New("inactive") },
			systemctlEnabled: func(context.Context, string) error { return errors.New("disabled") },
		}
		got, ok := eval.ServiceStatus(&model.DetectionSpec{SystemdUserUnits: []string{"sonarr@{user}.service"}}, "admin")
		assert.True(t, ok)
		assert.Equal(t, []string{"sonarr@admin.service"}, got.Units)
		assert.False(t, got.Active)
		assert.False(t, got.Enabled)
	})
}

type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "nginx" }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return 0o755 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return true }
func (fakeFileInfo) Sys() any           { return nil }
