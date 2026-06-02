//go:build linux

package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollector_Collect(t *testing.T) {
	t.Parallel()

	info, err := NewCollector().Collect()
	require.NoError(t, err)

	assert.NotEmpty(t, info.Hostname)
	assert.Positive(t, info.UptimeSeconds)
	assert.Positive(t, info.CPUCount)
	assert.NotZero(t, info.Memory.TotalBytes)
	require.NotEmpty(t, info.Disks)

	var root *DiskUsage
	for i := range info.Disks {
		if info.Disks[i].Mount == "/" {
			root = &info.Disks[i]
			break
		}
	}
	require.NotNil(t, root)
	assert.NotZero(t, root.TotalBytes)
}

func TestReadNetworkCounters(t *testing.T) {
	t.Parallel()

	counters, err := readNetworkCounters()
	require.NoError(t, err)
	// Cumulative counters; zero is valid on an idle fresh VM.
	_ = counters
}

func TestReadDiskIOCounters(t *testing.T) {
	t.Parallel()

	counters, err := readDiskIOCounters()
	require.NoError(t, err)
	_ = counters
}
