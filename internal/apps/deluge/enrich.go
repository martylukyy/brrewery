package deluge

import (
	"context"
	"errors"
	"strings"

	"github.com/autobrr/brrewery/internal/apps/extravars"
)

// EnrichAnsibleVars resolves the concrete Deluge release for the chosen version
// line and the libtorrent branch to build against, writing them into the Ansible
// extra vars. The deluge_build role reads the rest of the per-line build profile
// (Python runtime, C++ standard, setuptools pin, legacy toolchain) from the
// vendored manifest directly, and clones the chosen libtorrent branch head.
func EnrichAnsibleVars(ctx context.Context, vars map[string]string, resolver *ReleaseResolver) error {
	if resolver == nil {
		resolver = DefaultReleaseResolver()
	}

	version := strings.TrimSpace(vars[extravars.DelugeVersion])
	if version == "" {
		return errors.New("deluge_version is required")
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
	vars[extravars.DelugeVersion] = line.Version

	branch := strings.TrimSpace(vars[extravars.LibtorrentBranch])
	if branch == "" {
		branch = line.Libtorrent.Default
	}
	if !line.AllowsBranch(branch) {
		return ErrBranchNotAllowed
	}
	vars[extravars.LibtorrentBranch] = branch

	release, err := resolver.ResolveLatest(ctx, line.Series)
	if err != nil {
		return err
	}
	vars[extravars.DelugeRelease] = release
	return nil
}
