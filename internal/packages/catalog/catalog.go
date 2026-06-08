// Package catalog assembles the package catalog from embedded per-package
// manifests under manifests/.
//
// Adding a package to brrewery does NOT require editing this file. Drop a
// <id>.yaml manifest beside the others, add the install/upgrade/remove
// playbooks under ansible/playbooks/packages/<id>/, and an icon at
// web/public/packages/<id>.png. See docs/adding-a-package.md.
//
// The icon path (/packages/<id>.png) and the playbook paths are derived from the
// package id by convention. Packages whose install options must be computed at
// runtime (e.g. qBittorrent, whose version choices come from its build manifest)
// register a provider via RegisterInstallOptions from their own package instead
// of declaring static install_options.
package catalog

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/autobrr/brrewery/internal/packages/extravars"
	"github.com/autobrr/brrewery/internal/packages/model"
	"github.com/autobrr/brrewery/internal/paths"
)

//go:embed manifests/*.yaml
var manifestFS embed.FS

// manifest is the authored, declarative shape of a catalog entry. Fields not
// present here (icon path, playbook paths, runtime install options) are derived.
type manifest struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Category    string `yaml:"category"`
	WebPath     string `yaml:"web_path"`
	// Icon is an optional override for the icon basename. Defaults to "<id>.png".
	Icon         string              `yaml:"icon"`
	Dependencies []string            `yaml:"dependencies"`
	Detection    model.DetectionSpec `yaml:"detection"`
	// RequiresAccountPassword adds the shared account-password prompt (see
	// passwordSecret) to the package's install secrets.
	RequiresAccountPassword bool `yaml:"requires_account_password"`
	// InstallSecrets are additional, package-specific install-time prompts.
	InstallSecrets []model.InstallSecret `yaml:"install_secrets"`
	// InstallOptions are statically declared options. Packages that must compute
	// options at runtime register a provider via RegisterInstallOptions instead.
	InstallOptions []model.InstallOption `yaml:"install_options"`
}

// manifests holds the parsed catalog, sorted by display name. Parsed once at
// package initialization; a malformed or missing manifest is a build-time bug
// and panics loudly.
var manifests = mustLoadManifests()

func mustLoadManifests() []manifest {
	entries, err := fs.ReadDir(manifestFS, "manifests")
	if err != nil {
		panic(fmt.Sprintf("catalog: reading embedded manifests: %v", err))
	}

	out := make([]manifest, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := manifestFS.ReadFile("manifests/" + e.Name())
		if err != nil {
			panic(fmt.Sprintf("catalog: reading manifest %s: %v", e.Name(), err))
		}
		var m manifest
		if err := yaml.Unmarshal(data, &m); err != nil {
			panic(fmt.Sprintf("catalog: parsing manifest %s: %v", e.Name(), err))
		}
		if m.ID == "" {
			panic(fmt.Sprintf("catalog: manifest %s is missing an id", e.Name()))
		}
		if _, dup := seen[m.ID]; dup {
			panic(fmt.Sprintf("catalog: duplicate package id %q (%s)", m.ID, e.Name()))
		}
		seen[m.ID] = struct{}{}
		out = append(out, m)
	}
	if len(out) == 0 {
		panic("catalog: no package manifests embedded")
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func playbook(id, name string) string {
	return filepath.Join(paths.ResolveAnsibleRoot(), "playbooks", "packages", id, name+".yml")
}

// passwordSecret is the single install-time password prompt shared by every
// package that provisions a service account. It collects the operator's account
// password — the same value is the Linux user password, the sudo (become)
// password and the brrewery dashboard password — and is always verified against
// the brrewery account before install.
func passwordSecret() model.InstallSecret {
	return model.InstallSecret{
		Key:                    extravars.BecomePassword,
		Label:                  "Password",
		Type:                   "password",
		VerifyBrreweryPassword: true,
	}
}

func (m manifest) toPackage() model.Package {
	icon := m.Icon
	if icon == "" {
		icon = m.ID + ".png"
	}

	secrets := m.InstallSecrets
	if m.RequiresAccountPassword {
		secrets = append([]model.InstallSecret{passwordSecret()}, secrets...)
	}

	return model.Package{
		ID:             m.ID,
		Name:           m.Name,
		Description:    m.Description,
		Category:       m.Category,
		Icon:           "/packages/" + icon,
		WebPath:        m.WebPath,
		InstallSecrets: secrets,
		InstallOptions: installOptionsFor(m.ID, m.InstallOptions),
		Dependencies:   m.Dependencies,
		Detection:      m.Detection,
		Playbooks: model.PlaybookPaths{
			Install: playbook(m.ID, "install"),
			Upgrade: playbook(m.ID, "upgrade"),
			Remove:  playbook(m.ID, "remove"),
		},
	}
}

var (
	providersMu      sync.RWMutex
	optionsProviders = map[string]func() []model.InstallOption{}
)

// RegisterInstallOptions registers a provider that computes a package's install
// options at runtime. Most packages declare static install_options in their
// manifest and never need this; it exists for packages (e.g. qBittorrent) whose
// options derive from external state. Call it from an init function in the
// package that owns the logic.
func RegisterInstallOptions(id string, provider func() []model.InstallOption) {
	providersMu.Lock()
	defer providersMu.Unlock()
	optionsProviders[id] = provider
}

func installOptionsFor(id string, static []model.InstallOption) []model.InstallOption {
	providersMu.RLock()
	provider, ok := optionsProviders[id]
	providersMu.RUnlock()
	if ok {
		return provider()
	}
	return static
}

// All returns the package catalog.
func All() []model.Package {
	out := make([]model.Package, 0, len(manifests))
	for i := range manifests {
		out = append(out, manifests[i].toPackage())
	}
	return out
}

func ByID(id string) (model.Package, bool) {
	for i := range manifests {
		if manifests[i].ID == id {
			return manifests[i].toPackage(), true
		}
	}
	return model.Package{}, false
}

func DetectionSpec(id string) model.DetectionSpec {
	for i := range manifests {
		if manifests[i].ID == id {
			return manifests[i].Detection
		}
	}
	return model.DetectionSpec{}
}
