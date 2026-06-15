package rtorrent

import (
	"context"
	"errors"
	"strings"

	"github.com/autobrr/brrewery/internal/apps/extravars"
)

// EnrichAnsibleVars resolves the concrete rtorrent and libtorrent versions for
// the chosen version line and writes them into the Ansible extra vars. The
// rtorrent_build role reads the rest of the per-line build profile (source mode,
// C++ standard, rc dialect, patches) from the vendored manifest directly.
func EnrichAnsibleVars(ctx context.Context, vars map[string]string, resolver *ReleaseResolver) error {
	if resolver == nil {
		resolver = DefaultReleaseResolver()
	}

	version := strings.TrimSpace(vars[extravars.RtorrentVersion])
	if version == "" {
		return errors.New("rtorrent_version is required")
	}

	m, err := LoadManifest()
	if err != nil {
		return err
	}
	line, err := m.ResolveSelection(version)
	if err != nil {
		return err
	}
	// Normalize to the manifest's canonical line label.
	vars[extravars.RtorrentVersion] = line.Version

	rtorrentRelease, libtorrentRelease, err := resolver.ResolveVersions(ctx, line)
	if err != nil {
		return err
	}
	vars[extravars.RtorrentRelease] = rtorrentRelease
	vars[extravars.LibtorrentRelease] = libtorrentRelease
	return nil
}
