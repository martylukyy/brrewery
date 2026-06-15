package apps

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/autobrr/brrewery/internal/apps/ansible"
	"github.com/autobrr/brrewery/internal/apps/detect"
	"github.com/autobrr/brrewery/internal/apps/jobs"
	"github.com/autobrr/brrewery/internal/apps/model"
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

	playbookPath := filepath.Join("..", "..", "..", "ansible", "playbooks", "apps", "autobrr", "install.yml")
	store := jobs.NewStore()
	svc := NewServiceWithDeps(detect.NewEvaluator(), stubRunner{}, store)

	t.Run("unknown app", func(t *testing.T) {
		t.Parallel()
		_, err := svc.StartInstall(context.Background(), "missing", "admin", nil)
		require.ErrorIs(t, err, ErrAppNotFound)
	})

	t.Run("accepts install job", func(t *testing.T) {
		t.Parallel()

		svc := NewServiceWithDeps(detect.NewEvaluator(), stubRunner{}, jobs.NewStore())
		// Detection now keys off persistent artifacts (binary + unit file), so on
		// a host where autobrr is already installed StartInstall correctly refuses
		// with ErrAlreadyInstalled. This test exercises the not-installed path.
		if status, ok := svc.Get("autobrr", "admin"); ok && status.Installed {
			t.Skip("autobrr already installed on this host")
		}
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
