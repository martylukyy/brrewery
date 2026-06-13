package jobs

import (
	"path/filepath"
	"testing"

	"github.com/autobrr/brrewery/internal/apps/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreCreateGetLogs(t *testing.T) {
	t.Parallel()

	store := NewStore()
	job := store.Create("autobrr", model.JobActionInstall)
	require.NotEmpty(t, job.ID)

	got, ok := store.Get(job.ID)
	require.True(t, ok)
	assert.Equal(t, model.JobStatusQueued, got.Status)

	store.AppendLog(job.ID, "line one")
	store.MarkRunning(job.ID)
	store.SetStatus(job.ID, model.JobStatusSucceeded, "")

	logs, ok := store.Logs(job.ID)
	require.True(t, ok)
	assert.Equal(t, []string{"line one"}, logs)

	got, ok = store.Get(job.ID)
	require.True(t, ok)
	assert.Equal(t, model.JobStatusSucceeded, got.Status)
	assert.NotNil(t, got.FinishedAt)
}

func TestStorePersistsToDisk(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStoreAt(dir)
	job := store.Create("autobrr", model.JobActionInstall)
	store.AppendLog(job.ID, "persist me")
	store.MarkRunning(job.ID)

	reloaded := NewStoreAt(dir)
	got, ok := reloaded.Get(job.ID)
	require.True(t, ok)
	assert.Equal(t, model.JobStatusRunning, got.Status)

	logs, ok := reloaded.Logs(job.ID)
	require.True(t, ok)
	assert.Equal(t, []string{"persist me"}, logs)

	_, err := filepath.Abs(filepath.Join(dir, job.ID+".json"))
	require.NoError(t, err)
}
