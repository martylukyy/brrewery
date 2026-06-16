package deluge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/extravars"
)

func TestLoadManifestHasAllLines(t *testing.T) {
	m, err := LoadManifest()
	require.NoError(t, err)
	require.Len(t, m.Lines, 4)

	got := make([]string, 0, len(m.Lines))
	for _, l := range m.Lines {
		got = append(got, l.Version)
	}
	assert.Equal(t, []string{"2.2.x", "2.1.x", "2.0.x", "1.3.x"}, got)

	assert.Equal(t, "-O3 -mtune=native", m.Defaults.CompilerFlags)
	assert.Equal(t, "1_86_0", m.Defaults.Boost)
}

func TestResolveSelectionModernLine(t *testing.T) {
	m, err := LoadManifest()
	require.NoError(t, err)

	line, err := m.ResolveSelection("2.2.x")
	require.NoError(t, err)
	assert.Equal(t, "2.2", line.Series)
	assert.Equal(t, "python3", line.Python)
	assert.Equal(t, BranchRC12, line.Libtorrent.Default)
	assert.True(t, line.HasBranchChoice())
	assert.True(t, line.AllowsBranch(BranchRC12))
	assert.True(t, line.AllowsBranch(BranchRC20))
	assert.False(t, line.AllowsBranch(BranchRC11))
}

func TestResolveSelectionLegacyLine(t *testing.T) {
	m, err := LoadManifest()
	require.NoError(t, err)

	line, err := m.ResolveSelection("1.3.x")
	require.NoError(t, err)
	assert.Equal(t, "1.3", line.Series)
	assert.Equal(t, "python2.7", line.Python)
	assert.Equal(t, BranchRC11, line.Libtorrent.Default)
	assert.False(t, line.HasBranchChoice())
	assert.True(t, line.AllowsBranch(BranchRC11))
	assert.False(t, line.AllowsBranch(BranchRC20))
}

func TestResolveSelectionUnknown(t *testing.T) {
	m, err := LoadManifest()
	require.NoError(t, err)

	_, err = m.ResolveSelection("9.9.x")
	require.ErrorIs(t, err, ErrUnknownVersion)

	_, err = m.ResolveSelection("")
	require.ErrorIs(t, err, ErrUnknownVersion)
}

func TestInstallOptions(t *testing.T) {
	opts := InstallOptions()
	require.Len(t, opts, 2)

	assert.Equal(t, extravars.DelugeVersion, opts[0].Key)
	assert.Equal(t, "select", opts[0].Type)
	require.Len(t, opts[0].Choices, 4)
	assert.Equal(t, "2.2.x", opts[0].Choices[0].Value)
	assert.Equal(t, "1.3.x", opts[0].Choices[3].Value)

	// The libtorrent branch picker is shown only for the lines that offer a
	// choice (the python3 lines), gated via When.
	assert.Equal(t, extravars.LibtorrentBranch, opts[1].Key)
	require.NotNil(t, opts[1].When)
	assert.Equal(t, extravars.DelugeVersion, opts[1].When.Key)
	assert.Equal(t, []string{"2.2.x", "2.1.x", "2.0.x"}, opts[1].When.OneOf)
	require.Len(t, opts[1].Choices, 2)
	assert.Equal(t, BranchRC12, opts[1].Choices[0].Value)
	assert.Equal(t, BranchRC20, opts[1].Choices[1].Value)
}
