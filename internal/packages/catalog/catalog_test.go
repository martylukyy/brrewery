package catalog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/packages/extravars"
)

func TestAll_HasExpectedPackages(t *testing.T) {
	t.Parallel()

	pkgs := All()
	require.Len(t, pkgs, 14)

	ids := make(map[string]struct{}, len(pkgs))
	for _, pkg := range pkgs {
		ids[pkg.ID] = struct{}{}
		assert.NotEmpty(t, pkg.Name)
		assert.NotEmpty(t, pkg.Icon)
		assert.True(t, strings.HasPrefix(pkg.Icon, "/packages/"), "icon for %s should be a bundled asset path", pkg.ID)
		assert.True(t, strings.HasSuffix(pkg.Icon, ".png"), "icon for %s should be a PNG", pkg.ID)
		assert.Contains(t, pkg.Playbooks.Install, pkg.ID)
	}

	// autobrr and qBittorrent share the single account-password prompt: the same
	// key, type and always verified against the brrewery account.
	for _, id := range []string{"autobrr", "qbittorrent"} {
		pkg, ok := ByID(id)
		require.True(t, ok)
		require.Len(t, pkg.InstallSecrets, 1, "%s should declare the shared password secret", id)
		secret := pkg.InstallSecrets[0]
		assert.Equal(t, extravars.BecomePassword, secret.Key, "%s password secret key", id)
		assert.Equal(t, "password", secret.Type, "%s password secret type", id)
		assert.True(t, secret.VerifyBrreweryPassword, "%s password must be verified", id)
	}

	for _, want := range []string{
		"qbittorrent", "autobrr", "sonarr", "radarr", "prowlarr",
		"lidarr", "sabnzbd", "deluge", "rtorrent", "rutorrent",
		"jellyfin", "plex", "filebrowser", "emby",
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
