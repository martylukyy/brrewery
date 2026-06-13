// Package qbittorrent loads the vendored build manifest and validates the
// install options (qBittorrent version, libtorrent branch, optional libtorrent
// patch) supplied by the API.
package qbittorrent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/autobrr/brrewery/internal/paths"
)

// AppID is the catalog id this validator applies to.
const AppID = "qbittorrent"

// Libtorrent branch identifiers.
const (
	BranchRC12 = "RC_1_2"
	BranchRC20 = "RC_2_0"
)

// Manifest mirrors ansible/roles/qbittorrent_build/files/qbittorrent/manifest.yml.
type Manifest struct {
	Defaults Defaults `yaml:"defaults"`
	Lines    []Line   `yaml:"lines"`
}

// Defaults holds dependency versions shared across all lines.
type Defaults struct {
	// BoostRC12 caps Boost for libtorrent RC_1_2 (Boost >= 1.87 drops io_service.hpp).
	BoostRC12     string `yaml:"boost_rc_1_2"`
	CompilerFlags string `yaml:"compiler_flags"`
}

// Line is the build profile for one qBittorrent release line (e.g. 5.2).
type Line struct {
	Version     string         `yaml:"version"`
	CxxStd      string         `yaml:"cxx_std"`
	BuildSystem string         `yaml:"build_system"`
	Qt          QtSpec         `yaml:"qt"`
	Libtorrent  LibtorrentSpec `yaml:"libtorrent"`
}

// QtSpec pins the minimum Qt version for a line. The install playbook resolves
// the newest compatible patch from download.qt.io at build time unless Version
// overrides it.
type QtSpec struct {
	Min     string `yaml:"min"`
	Version string `yaml:"version,omitempty"`
}

// QtVersionOverride returns a fixed Qt version when the manifest pins one.
func (l Line) QtVersionOverride() string {
	return strings.TrimSpace(l.Qt.Version)
}

// LibtorrentSpec lists the branches a line can build against.
type LibtorrentSpec struct {
	Default  string                          `yaml:"default"`
	Branches map[string]LibtorrentBranchSpec `yaml:"branches"`
}

// LibtorrentBranchSpec pins the libtorrent tag for a branch.
type LibtorrentBranchSpec struct {
	Tag string `yaml:"tag"`
}

var (
	manifestOnce sync.Once
	manifest     *Manifest
	manifestErr  error
)

// LoadManifest reads and caches the vendored manifest from the qBittorrent
// vendor root.
func LoadManifest() (*Manifest, error) {
	manifestOnce.Do(func() {
		manifest, manifestErr = loadManifestFrom(paths.ResolveVendorQBittorrentRoot())
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
		return nil, fmt.Errorf("%w: no qBittorrent lines defined", ErrManifestUnavailable)
	}
	return &m, nil
}

// LineForVersion returns the build profile for a release line (e.g. "5.2").
func (m *Manifest) LineForVersion(version string) (Line, bool) {
	for _, line := range m.Lines {
		if line.Version == version {
			return line, true
		}
	}
	return Line{}, false
}

// ResolveSelection maps the install UI version line to a manifest line.
func (m *Manifest) ResolveSelection(version string) (Line, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return Line{}, fmt.Errorf("%w: empty version line", ErrUnknownVersion)
	}
	line, ok := m.LineForVersion(version)
	if !ok {
		return Line{}, fmt.Errorf("%w: %q", ErrUnknownVersion, version)
	}
	return line, nil
}

// AllowsBranch reports whether the line can build against the given libtorrent branch.
func (l *Line) AllowsBranch(branch string) bool {
	_, ok := l.Libtorrent.Branches[branch]
	return ok
}
