// Package catalog assembles the app catalog from embedded per-app
// manifests under manifests/.
//
// Adding an app to brrewery does NOT require editing this file. Drop a
// <id>.yaml manifest beside the others, add the install/upgrade/remove
// playbooks under ansible/playbooks/apps/<id>/, and an icon at
// web/public/apps/<id>.png. See docs/adding-an-app.md.
//
// The icon path (/apps/<id>.png) and the playbook paths are derived from the
// app id by convention. Apps whose install options must be computed at
// runtime (e.g. qBittorrent, whose version choices come from its build manifest)
// register a provider via RegisterInstallOptions from their own app instead
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

	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/apps/model"
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
	// passwordSecret) to the app's install secrets.
	RequiresAccountPassword bool `yaml:"requires_account_password"`
	// InstallSecrets are additional, app-specific install-time prompts.
	InstallSecrets []model.InstallSecret `yaml:"install_secrets"`
	// InstallOptions are statically declared options. Apps that must compute
	// options at runtime register a provider via RegisterInstallOptions instead.
	InstallOptions []model.InstallOption `yaml:"install_options"`
}

// manifests holds the parsed catalog, sorted by display name. Parsed once at
// app initialization; a malformed or missing manifest is a build-time bug
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
			panic(fmt.Sprintf("catalog: duplicate app id %q (%s)", m.ID, e.Name()))
		}
		seen[m.ID] = struct{}{}
		out = append(out, m)
	}
	if len(out) == 0 {
		panic("catalog: no app manifests embedded")
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func playbook(id, name string) string {
	return filepath.Join(paths.ResolveAnsibleRoot(), "playbooks", "apps", id, name+".yml")
}

// passwordSecret is the single install-time password prompt shared by every
// app that provisions a service account. It collects the operator's account
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

func (m manifest) toApp() model.App {
	icon := m.Icon
	if icon == "" {
		icon = m.ID + ".png"
	}

	secrets := m.InstallSecrets
	if m.RequiresAccountPassword {
		secrets = append([]model.InstallSecret{passwordSecret()}, secrets...)
	}

	return model.App{
		ID:             m.ID,
		Name:           m.Name,
		Description:    m.Description,
		Category:       m.Category,
		Icon:           "/apps/" + icon,
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

// RegisterInstallOptions registers a provider that computes an app's install
// options at runtime. Most apps declare static install_options in their
// manifest and never need this; it exists for apps (e.g. qBittorrent) whose
// options derive from external state. Call it from an init function in the
// app that owns the logic.
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

// All returns the app catalog.
func All() []model.App {
	out := make([]model.App, 0, len(manifests))
	for i := range manifests {
		out = append(out, manifests[i].toApp())
	}
	return out
}

func ByID(id string) (model.App, bool) {
	for i := range manifests {
		if manifests[i].ID == id {
			return manifests[i].toApp(), true
		}
	}
	return model.App{}, false
}

func DetectionSpec(id string) model.DetectionSpec {
	for i := range manifests {
		if manifests[i].ID == id {
			return manifests[i].Detection
		}
	}
	return model.DetectionSpec{}
}
