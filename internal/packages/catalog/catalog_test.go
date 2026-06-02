package catalog

import (
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
		assert.Contains(t, pkg.Playbooks.Install, pkg.ID)
	}

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
