// Package rtorrent loads the vendored build manifest and resolves the concrete
// rtorrent + libtorrent versions for the release line chosen in the install UI.
// It mirrors the qBittorrent package: the install options (a version picker) are
// registered as a catalog options provider, and the resolved versions are added
// to the Ansible extra vars before the rtorrent_build role runs.
package rtorrent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/autobrr/brrewery/internal/paths"
)

// AppID is the catalog id this package applies to.
const AppID = "rtorrent"

// Source modes for fetching a line's sources.
const (
	// SourceRelease downloads the upstream dist tarballs (rtorrent + libtorrent)
	// from the GitHub release; they ship a pre-generated ./configure.
	SourceRelease = "release"
	// SourceGitArchive downloads the tag source archive and bootstraps it with
	// autoreconf (used for 0.9.6, which has no GitHub release).
	SourceGitArchive = "git-archive"
)

// Resolve modes for picking a line's concrete version.
const (
	// ResolveLatest picks the newest patch tag in the series from GitHub.
	ResolveLatest = "latest"
	// ResolvePinned uses the manifest tag verbatim.
	ResolvePinned = "pinned"
)

var (
	// ErrManifestUnavailable indicates the vendored manifest could not be read.
	ErrManifestUnavailable = errors.New("rtorrent build manifest unavailable")
	// ErrUnknownVersion indicates the requested version line is not in the manifest.
	ErrUnknownVersion = errors.New("unknown rtorrent version line")
)

// Manifest mirrors ansible/roles/rtorrent_build/files/rtorrent/manifest.yml.
type Manifest struct {
	Defaults Defaults `yaml:"defaults"`
	Lines    []Line   `yaml:"lines"`
}

// Defaults holds build settings shared across lines.
type Defaults struct {
	// LegacyCxxflags are appended for lines flagged legacy (C++11-era sources).
	LegacyCxxflags string `yaml:"legacy_cxxflags"`
}

// Line is the build profile for one selectable rtorrent release line.
type Line struct {
	// Version is the UI label and selection key (e.g. "0.16.x", "0.9.6").
	Version string `yaml:"version"`
	// Series is the major.minor prefix used to resolve the latest patch ("0.16").
	Series string `yaml:"series"`
	// Source is SourceRelease or SourceGitArchive.
	Source string `yaml:"source"`
	// Resolve is ResolveLatest or ResolvePinned.
	Resolve string `yaml:"resolve"`
	// Tag pins the rtorrent git tag for ResolvePinned lines (e.g. "v0.10.0").
	Tag string `yaml:"tag"`
	// LibtorrentTag pins the libtorrent tag for git-archive lines (e.g. "v0.13.6").
	LibtorrentTag string `yaml:"libtorrent_tag"`
	// CxxStd is the C++ standard the line must build with (e.g. "c++20").
	CxxStd string `yaml:"cxx_std"`
	// RcSyntax is the .rtorrent.rc dialect: "modern" or "legacy".
	RcSyntax string `yaml:"rc_syntax"`
	// Legacy marks C++11-era lines that need the legacy CXXFLAGS.
	Legacy bool `yaml:"legacy"`
	// Patches lists vendored patch basenames keyed by component ("libtorrent",
	// "rtorrent"); applied by the build role.
	Patches map[string][]string `yaml:"patches"`
}

var (
	manifestOnce sync.Once
	manifest     *Manifest
	manifestErr  error
)

// LoadManifest reads and caches the vendored manifest from the rtorrent vendor root.
func LoadManifest() (*Manifest, error) {
	manifestOnce.Do(func() {
		manifest, manifestErr = loadManifestFrom(paths.ResolveVendorRtorrentRoot())
	})
	return manifest, manifestErr
}

func loadManifestFrom(root string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(root, "manifest.yml"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrManifestUnavailable, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrManifestUnavailable, err)
	}
	if len(m.Lines) == 0 {
		return nil, fmt.Errorf("%w: no rtorrent lines defined", ErrManifestUnavailable)
	}
	return &m, nil
}

// ResolveSelection returns the build profile for the chosen version line.
func (m *Manifest) ResolveSelection(version string) (Line, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return Line{}, fmt.Errorf("%w: empty version line", ErrUnknownVersion)
	}
	for _, line := range m.Lines {
		if line.Version == version {
			return line, nil
		}
	}
	return Line{}, fmt.Errorf("%w: %q", ErrUnknownVersion, version)
}

// stripV removes a leading "v" from a git tag, yielding a bare version string.
func stripV(tag string) string {
	return strings.TrimPrefix(strings.TrimSpace(tag), "v")
}
