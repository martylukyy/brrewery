package detect

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/brrewery/internal/packages/model"
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
			},
			want: false,
		},
		{
			name: "unit active",
			spec: model.DetectionSpec{SystemdUnits: []string{"nginx.service"}},
			eval: &Evaluator{
				systemctlActive: func(context.Context, string) error { return nil },
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
			name: "user scoped unit active",
			spec: model.DetectionSpec{
				Binaries:         []string{"autobrr"},
				SystemdUserUnits: []string{"autobrr@{user}.service"},
			},
			eval: &Evaluator{
				lookPath: func(string) (string, error) { return "/usr/local/bin/autobrr", nil },
				systemctlEnabled: func(_ context.Context, unit string) error {
					if unit == "autobrr@admin.service" {
						return nil
					}
					return errors.New("not enabled")
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.eval.Installed(&tt.spec)
			if tt.name == "user scoped unit active" {
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
		systemctlEnabled: func(context.Context, string) error { return nil },
	}
	spec := model.DetectionSpec{
		Binaries:         []string{"autobrr"},
		SystemdUserUnits: []string{"autobrr@{user}.service"},
	}

	assert.False(t, eval.InstalledForUser(&spec, ""))
	assert.True(t, eval.InstalledForUser(&spec, "admin"))
}

type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "nginx" }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return 0o755 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return true }
func (fakeFileInfo) Sys() any           { return nil }
