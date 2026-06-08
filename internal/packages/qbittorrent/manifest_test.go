package qbittorrent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/packages/qbittorrent"
)

func TestLine_QtVersionOverride(t *testing.T) {
	t.Parallel()

	line := qbittorrent.Line{Qt: qbittorrent.QtSpec{Min: "6.6", Version: "6.8.2"}}
	assert.Equal(t, "6.8.2", line.QtVersionOverride())

	line = qbittorrent.Line{Qt: qbittorrent.QtSpec{Min: "6.6"}}
	assert.Empty(t, line.QtVersionOverride())
}

func TestLoadManifest_qtMinPerLine(t *testing.T) {
	t.Parallel()

	m, err := qbittorrent.LoadManifest()
	require.NoError(t, err)

	line, ok := m.LineForVersion("5.2")
	require.True(t, ok)
	assert.Equal(t, "6.6", line.Qt.Min)
	assert.Empty(t, line.QtVersionOverride())
}
