package packages

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/autobrr/brrewery/internal/packages/ansible"
	"github.com/autobrr/brrewery/internal/packages/detect"
	"github.com/autobrr/brrewery/internal/packages/jobs"
	"github.com/autobrr/brrewery/internal/packages/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRunner struct {
	err error
}

func (s stubRunner) Run(_ context.Context, req ansible.RunRequest) error {
	if req.OnOutput != nil {
		req.OnOutput("PLAY [localhost]")
	}
	return s.err
}

func TestService_StartInstall(t *testing.T) {
	t.Parallel()

	playbookPath := filepath.Join("..", "..", "..", "ansible", "playbooks", "packages", "autobrr", "install.yml")
	store := jobs.NewStore()
	svc := NewServiceWithDeps(detect.NewEvaluator(), stubRunner{}, store)

	t.Run("unknown package", func(t *testing.T) {
		t.Parallel()
		_, err := svc.StartInstall(context.Background(), "missing", "admin", nil)
		require.ErrorIs(t, err, ErrPackageNotFound)
	})

	t.Run("accepts install job", func(t *testing.T) {
		t.Parallel()

		svc := NewServiceWithDeps(detect.NewEvaluator(), stubRunner{}, jobs.NewStore())
		job, err := svc.StartInstall(context.Background(), "autobrr", "admin", nil)
		if err != nil {
			if err == ErrPlaybookMissing {
				t.Skip("autobrr playbook not available in test environment")
			}
			require.NoError(t, err)
		}

		require.NotEmpty(t, job.ID)

		require.Eventually(t, func() bool {
			got, ok := svc.GetJob(job.ID)
			return ok && got.Status == model.JobStatusFailed
		}, 2*time.Second, 20*time.Millisecond)

		got, ok := svc.GetJob(job.ID)
		require.True(t, ok)
		assert.Contains(t, got.Error, "not detected")

		logs, ok := svc.JobLogs(job.ID)
		require.True(t, ok)
		assert.NotEmpty(t, logs)
	})

	_ = playbookPath
}
