package paths_test

import (
	"testing"

	"github.com/autobrr/brrewery/internal/paths"
	"github.com/stretchr/testify/assert"
)

func TestResolveAnsibleRootFromEnv(t *testing.T) {
	t.Setenv("BRREWERY_ANSIBLE_ROOT", "/tmp/custom-ansible")
	assert.Equal(t, "/tmp/custom-ansible", paths.ResolveAnsibleRoot())
}

func TestListenAddressFromEnv(t *testing.T) {
	t.Setenv("BRREWERY_LISTEN_ADDR", "127.0.0.1:9090")
	assert.Equal(t, "127.0.0.1:9090", paths.ListenAddress())
}

func TestResolveJobsDirFromEnv(t *testing.T) {
	t.Setenv("BRREWERY_JOBS_DIR", "/tmp/brrewery-jobs")
	assert.Equal(t, "/tmp/brrewery-jobs", paths.ResolveJobsDir())
}
