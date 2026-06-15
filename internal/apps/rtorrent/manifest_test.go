package rtorrent

import (
	"testing"

	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadManifestHasAllLines(t *testing.T) {
	m, err := LoadManifest()
	require.NoError(t, err)
	require.Len(t, m.Lines, 5)

	got := make([]string, 0, len(m.Lines))
	for _, l := range m.Lines {
		got = append(got, l.Version)
	}
	assert.Equal(t, []string{"0.16.x", "0.15.x", "0.10.0", "0.9.8", "0.9.6"}, got)
}

func TestResolveSelection(t *testing.T) {
	m, err := LoadManifest()
	require.NoError(t, err)

	line, err := m.ResolveSelection("0.9.6")
	require.NoError(t, err)
	assert.Equal(t, SourceGitArchive, line.Source)
	assert.Equal(t, "v0.13.6", line.LibtorrentTag)
	assert.Equal(t, "legacy", line.RcSyntax)
	assert.True(t, line.Legacy)
	assert.Equal(t, []string{"libtorrent-0.13.6-openssl.patch"}, line.Patches["libtorrent"])

	latest, err := m.ResolveSelection("0.16.x")
	require.NoError(t, err)
	assert.Equal(t, ResolveLatest, latest.Resolve)
	assert.Equal(t, "0.16", latest.Series)
	assert.Equal(t, "c++20", latest.CxxStd)

	_, err = m.ResolveSelection("9.9.9")
	assert.ErrorIs(t, err, ErrUnknownVersion)

	_, err = m.ResolveSelection("")
	assert.ErrorIs(t, err, ErrUnknownVersion)
}

func TestInstallOptions(t *testing.T) {
	opts := InstallOptions()
	require.Len(t, opts, 1)
	assert.Equal(t, extravars.RtorrentVersion, opts[0].Key)
	assert.Equal(t, "select", opts[0].Type)
	require.Len(t, opts[0].Choices, 5)
	assert.Equal(t, "0.16.x", opts[0].Choices[0].Value)
	assert.Equal(t, "0.9.6", opts[0].Choices[4].Value)
}

func TestStripV(t *testing.T) {
	assert.Equal(t, "0.16.14", stripV("v0.16.14"))
	assert.Equal(t, "0.13.6", stripV(" v0.13.6 "))
	assert.Equal(t, "0.10.0", stripV("0.10.0"))
}
