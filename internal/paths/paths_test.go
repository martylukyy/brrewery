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

func TestResolveJobsDirDefault(t *testing.T) {
	t.Setenv("BRREWERY_JOBS_DIR", "")
	assert.Equal(t, paths.JobsDir, paths.ResolveJobsDir())
}

func TestResolveJobsDirFromEnv(t *testing.T) {
	t.Setenv("BRREWERY_JOBS_DIR", "/tmp/brrewery-jobs")
	assert.Equal(t, "/tmp/brrewery-jobs", paths.ResolveJobsDir())
}

func TestResolveJobsDirInMemory(t *testing.T) {
	t.Setenv("BRREWERY_JOBS_DIR", "-")
	assert.Empty(t, paths.ResolveJobsDir())
}

func TestResolveVendorQBittorrentRootFromEnv(t *testing.T) {
	t.Setenv("BRREWERY_QBITTORRENT_VENDOR_ROOT", "/tmp/custom-qbt-vendor")
	assert.Equal(t, "/tmp/custom-qbt-vendor", paths.ResolveVendorQBittorrentRoot())
}
