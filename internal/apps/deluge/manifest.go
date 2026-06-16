// Package deluge loads the vendored build manifest and resolves the concrete
// Deluge release (and libtorrent line) chosen in the install UI. It mirrors the
// rtorrent and qBittorrent packages: the install options (a version picker plus
// a libtorrent branch picker for the Python 3 lines) are registered as a catalog
// options provider, and the resolved versions are added to the Ansible extra
// vars before the deluge_build role runs.
package deluge

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
const AppID = "deluge"

// Libtorrent branch identifiers. RC_1_1 is the last branch with Python 2
// bindings (Deluge 1.3); RC_1_2 and RC_2_0 are the Python 3 choices.
const (
	BranchRC11 = "RC_1_1"
	BranchRC12 = "RC_1_2"
	BranchRC20 = "RC_2_0"
)

var (
	// ErrManifestUnavailable indicates the vendored manifest could not be read.
	ErrManifestUnavailable = errors.New("deluge build manifest unavailable")
	// ErrUnknownVersion indicates the requested version line is not in the manifest.
	ErrUnknownVersion = errors.New("unknown deluge version line")
	// ErrBranchNotAllowed indicates the libtorrent branch is not valid for the line.
	ErrBranchNotAllowed = errors.New("libtorrent branch not supported for this Deluge version")
)

// Manifest mirrors ansible/roles/deluge_build/files/deluge/manifest.yml.
type Manifest struct {
	Defaults Defaults `yaml:"defaults"`
	Lines    []Line   `yaml:"lines"`
}

// Defaults holds build settings shared across lines. Go only reads what it
// surfaces; the deluge_build role reads the rest of the defaults directly.
type Defaults struct {
	CompilerFlags string `yaml:"compiler_flags"`
	Boost         string `yaml:"boost"`
}

// Line is the build profile for one selectable Deluge release line.
type Line struct {
	// Version is the UI label and selection key (e.g. "2.2.x", "1.3.x").
	Version string `yaml:"version"`
	// Series is the major.minor prefix used to resolve the latest Deluge tag.
	Series string `yaml:"series"`
	// Python is the interpreter the line builds and runs under ("python3"/"python2.7").
	Python string `yaml:"python"`
	// Libtorrent lists the branches this line can build against.
	Libtorrent LibtorrentSpec `yaml:"libtorrent"`
}

// LibtorrentSpec lists the libtorrent branches a line allows and its default.
type LibtorrentSpec struct {
	Default  string   `yaml:"default"`
	Branches []string `yaml:"branches"`
}

// AllowsBranch reports whether the line can build against the given branch.
func (l *Line) AllowsBranch(branch string) bool {
	for _, b := range l.Libtorrent.Branches {
		if b == branch {
			return true
		}
	}
	return false
}

// HasBranchChoice reports whether the line offers more than one libtorrent branch.
func (l *Line) HasBranchChoice() bool {
	return len(l.Libtorrent.Branches) > 1
}

var (
	manifestOnce sync.Once
	manifest     *Manifest
	manifestErr  error
)

// LoadManifest reads and caches the vendored manifest from the deluge vendor root.
func LoadManifest() (*Manifest, error) {
	manifestOnce.Do(func() {
		manifest, manifestErr = loadManifestFrom(paths.ResolveVendorDelugeRoot())
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
		return nil, fmt.Errorf("%w: no deluge lines defined", ErrManifestUnavailable)
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
