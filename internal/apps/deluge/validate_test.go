package deluge_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/deluge"
	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/apps/libtorrent"
)

func TestValidate_NonDelugeAppPasses(t *testing.T) {
	t.Parallel()

	err := deluge.Validate("autobrr", map[string]string{"anything": "goes"})
	require.NoError(t, err)
}

func TestValidate_DelugeChecksLibtorrentPatch(t *testing.T) {
	t.Parallel()

	m, err := deluge.LoadManifest()
	require.NoError(t, err)
	require.NotEmpty(t, m.Lines)
	version := m.Lines[0].Version

	validDiff := base64.StdEncoding.EncodeToString(
		[]byte("--- a/src/settings_pack.cpp\n+++ b/src/settings_pack.cpp\n@@ -1 +1 @@\n-old\n+new\n"),
	)

	// A well-formed patch is accepted alongside a valid version.
	err = deluge.Validate(deluge.AppID, map[string]string{
		extravars.DelugeVersion:   version,
		extravars.LibtorrentPatch: validDiff,
	})
	require.NoError(t, err)

	// A malformed patch is rejected at the API rather than failing mid-build.
	err = deluge.Validate(deluge.AppID, map[string]string{
		extravars.DelugeVersion:   version,
		extravars.LibtorrentPatch: "not-base64-$$$",
	})
	require.ErrorIs(t, err, libtorrent.ErrPatchInvalid)

	// An absent patch stays valid: the build falls back to the operator/vendored tier.
	err = deluge.Validate(deluge.AppID, map[string]string{
		extravars.DelugeVersion: version,
	})
	require.NoError(t, err)
}
