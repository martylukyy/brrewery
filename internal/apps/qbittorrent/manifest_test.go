package qbittorrent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/qbittorrent"
)

func TestLoadManifest_pinsBuildDependenciesPerLine(t *testing.T) {
	t.Parallel()

	m, err := qbittorrent.LoadManifest()
	require.NoError(t, err)

	line, ok := m.LineForVersion("5.2")
	require.True(t, ok)
	assert.Equal(t, "6.11.1", line.Qt)
	assert.Equal(t, "1.3.2", line.Zlib)
	assert.Equal(t, "3.6.3", line.Openssl)
	assert.Equal(t, "-O3 -mtune=native", line.CompilerFlags)
	assert.Equal(t, "1_86_0", line.Libtorrent.Branches["RC_1_2"].Boost)
	assert.Equal(t, "1_91_0", line.Libtorrent.Branches["RC_2_0"].Boost)
}

// 5.3 follows upstream's official builds: Qt 6.11.2 with libtorrent 1.2.x or
// 2.1.x (no 2.0.x).
func TestLoadManifest_line53MatchesUpstreamBuilds(t *testing.T) {
	t.Parallel()

	m, err := qbittorrent.LoadManifest()
	require.NoError(t, err)

	line, ok := m.LineForVersion("5.3")
	require.True(t, ok)
	assert.Equal(t, "6.11.2", line.Qt)
	assert.Equal(t, qbittorrent.BranchRC12, line.Libtorrent.Default)
	assert.Equal(t, qbittorrent.LibtorrentBranchSpec{Tag: "v1.2.20", Boost: "1_86_0"}, line.Libtorrent.Branches[qbittorrent.BranchRC12])
	assert.Equal(t, qbittorrent.LibtorrentBranchSpec{Tag: "v2.1.2", Boost: "1_92_0"}, line.Libtorrent.Branches[qbittorrent.BranchRC21])
	assert.False(t, line.AllowsBranch(qbittorrent.BranchRC20))
}
