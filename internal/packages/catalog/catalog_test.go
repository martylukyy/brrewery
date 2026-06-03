package catalog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAll_HasExpectedPackages(t *testing.T) {
	t.Parallel()

	pkgs := All()
	require.Len(t, pkgs, 16)

	ids := make(map[string]struct{}, len(pkgs))
	for _, pkg := range pkgs {
		ids[pkg.ID] = struct{}{}
		assert.NotEmpty(t, pkg.Name)
		assert.NotEmpty(t, pkg.Icon)
		assert.True(t, strings.HasPrefix(pkg.Icon, "/packages/"), "icon for %s should be a bundled asset path", pkg.ID)
		assert.True(t, strings.HasSuffix(pkg.Icon, ".png"), "icon for %s should be a PNG", pkg.ID)
		assert.Contains(t, pkg.Playbooks.Install, pkg.ID)
	}

	autobrr, ok := ByID("autobrr")
	require.True(t, ok)
	require.Len(t, autobrr.InstallSecrets, 1)
	assert.True(t, autobrr.InstallSecrets[0].VerifyBrreweryPassword)

	for _, want := range []string{
		"qbittorrent", "autobrr", "sonarr", "radarr", "prowlarr",
		"lidarr", "bazarr", "sabnzbd", "deluge", "rtorrent", "rutorrent",
		"jellyfin", "plex", "organizr", "filebrowser", "emby",
	} {
		_, ok := ids[want]
		assert.True(t, ok, "missing package %s", want)
	}
}

func TestByID(t *testing.T) {
	t.Parallel()

	pkg, ok := ByID("sonarr")
	require.True(t, ok)
	assert.Equal(t, "Sonarr", pkg.Name)

	_, ok = ByID("nonexistent")
	assert.False(t, ok)
}
