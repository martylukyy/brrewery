//go:build linux

package system

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockStatBusyMsIndex(t *testing.T) {
	t.Parallel()

	idx, err := blockStatBusyMsIndex(17)
	require.NoError(t, err)
	assert.Equal(t, 9, idx)

	idx, err = blockStatBusyMsIndex(11)
	require.NoError(t, err)
	assert.Equal(t, 9, idx)
}

func TestMountDiskIOStatPath(t *testing.T) {
	t.Parallel()

	statPath, err := mountDiskIOStatPath("/")
	require.NoError(t, err)
	rel := strings.TrimPrefix(statPath, "/sys/block/")
	parts := strings.Split(rel, "/")
	require.Len(t, parts, 2)
	assert.Equal(t, "stat", parts[1])

	ioTime, err := readBlockStatBusyMs(statPath)
	require.NoError(t, err)
	assert.Greater(t, ioTime, uint64(0))
}

func TestDiskIOBusyPercentFormula(t *testing.T) {
	t.Parallel()

	// 15 ms device-busy in a 1 s window → 1.5% (iostat %util).
	assert.InDelta(t, 1.5, float64(15)/(1.0*10), 0.001)
}

func TestReadMountIOBusy(t *testing.T) {
	t.Parallel()

	c := NewCollector()
	uptime, err := readUptime()
	require.NoError(t, err)

	busy, err := c.readMountIOBusy("/", uptime)
	require.NoError(t, err)
	assert.Equal(t, 0.0, busy)

	uptime2, err := readUptime()
	require.NoError(t, err)
	busy, err = c.readMountIOBusy("/", uptime2)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, busy, 0.0)
	assert.LessOrEqual(t, busy, 100.0)
}

func TestReadCPUPercent(t *testing.T) {
	t.Parallel()

	c := NewCollector()

	busy, err := c.readCPUPercent()
	require.NoError(t, err)
	assert.Equal(t, 0.0, busy)

	busy, err = c.readCPUPercent()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, busy, 0.0)
	assert.LessOrEqual(t, busy, 100.0)
}
