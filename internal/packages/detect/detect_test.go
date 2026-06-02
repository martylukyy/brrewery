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
				systemctl: func(context.Context, string) error { return nil },
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.eval.Installed(&tt.spec)
			assert.Equal(t, tt.want, got)
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
