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
	assert.NotZero(t, info.Disk.TotalBytes)
	assert.Equal(t, "/", info.Disk.Mount)
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
