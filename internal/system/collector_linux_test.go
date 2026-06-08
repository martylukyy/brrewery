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

func TestDiskUsageFromStatfs(t *testing.T) {
	t.Parallel()

	const bsize = 4096

	// 100 blocks total: 10 used, 90 free, of which 5 are root-reserved.
	// Bfree counts all free blocks (90); Bavail omits the reserved ones (85).
	usage := diskUsageFromStatfs(100, 90, 85, bsize)

	assert.Equal(t, uint64(100*bsize), usage.TotalBytes)
	// Used is derived from Bfree (blocks-90), not Bavail (blocks-85): the
	// reserved-but-free blocks must not be counted as used.
	assert.Equal(t, uint64(10*bsize), usage.UsedBytes)
	assert.Equal(t, uint64(85*bsize), usage.AvailableBytes)
	// df Use%: used / (used + available) = 10 / 95, reserved excluded from basis.
	assert.InDelta(t, 100*10.0/95.0, usage.UsedPercent, 0.001)

	// Empty filesystem: nothing used, no division by zero.
	empty := diskUsageFromStatfs(100, 100, 100, bsize)
	assert.Equal(t, uint64(0), empty.UsedBytes)
	assert.Equal(t, 0.0, empty.UsedPercent)
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
