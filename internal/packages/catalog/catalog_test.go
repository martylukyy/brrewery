package catalog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/packages/extravars"
	"github.com/autobrr/brrewery/internal/packages/model"
)

// TestAll_Invariants validates the contract every manifest must satisfy. It is
// intentionally data-driven: adding a package (a new manifest) requires no
// change here.
func TestAll_Invariants(t *testing.T) {
	t.Parallel()

	pkgs := All()
	require.NotEmpty(t, pkgs, "catalog should not be empty")

	ids := make(map[string]struct{}, len(pkgs))
	for _, pkg := range pkgs {
		assert.NotEmpty(t, pkg.ID, "package id must be set")
		_, dup := ids[pkg.ID]
		assert.False(t, dup, "duplicate package id %q", pkg.ID)
		ids[pkg.ID] = struct{}{}

		assert.NotEmpty(t, pkg.Name, "%s: name must be set", pkg.ID)
		assert.NotEmpty(t, pkg.Description, "%s: description must be set", pkg.ID)
		assert.NotEmpty(t, pkg.Category, "%s: category must be set", pkg.ID)

		assert.True(t, strings.HasPrefix(pkg.Icon, "/packages/"), "%s: icon should be a bundled asset path", pkg.ID)
		assert.True(t, strings.HasSuffix(pkg.Icon, ".png"), "%s: icon should be a PNG", pkg.ID)

		assert.Contains(t, pkg.Playbooks.Install, pkg.ID, "%s: install playbook path", pkg.ID)
		assert.Contains(t, pkg.Playbooks.Upgrade, pkg.ID, "%s: upgrade playbook path", pkg.ID)
		assert.Contains(t, pkg.Playbooks.Remove, pkg.ID, "%s: remove playbook path", pkg.ID)

		assert.True(t, hasDetection(pkg.Detection), "%s: detection must declare at least one check", pkg.ID)

		assertSharedPasswordContract(t, pkg)
	}
}

// hasDetection reports whether a detection spec declares at least one check, so
// a package can never be reported as installed purely by default.
func hasDetection(d model.DetectionSpec) bool {
	return len(d.Binaries) > 0 ||
		len(d.SystemdUnits) > 0 ||
		len(d.SystemdUserUnits) > 0 ||
		len(d.Paths) > 0
}

// assertSharedPasswordContract enforces that any verified secret is the shared
// account-password prompt: same key and type, regardless of which package
// declares it.
func assertSharedPasswordContract(t *testing.T, pkg model.Package) {
	t.Helper()
	for _, secret := range pkg.InstallSecrets {
		if !secret.VerifyBrreweryPassword {
			continue
		}
		assert.Equal(t, extravars.BecomePassword, secret.Key, "%s: verified secret key", pkg.ID)
		assert.Equal(t, "password", secret.Type, "%s: verified secret type", pkg.ID)
	}
}

// TestRequiresAccountPassword confirms the manifest shorthand expands to the
// shared, verified password secret.
func TestRequiresAccountPassword(t *testing.T) {
	t.Parallel()

	pkg, ok := ByID("qui")
	require.True(t, ok)
	require.Len(t, pkg.InstallSecrets, 1)
	secret := pkg.InstallSecrets[0]
	assert.Equal(t, extravars.BecomePassword, secret.Key)
	assert.Equal(t, "password", secret.Type)
	assert.True(t, secret.VerifyBrreweryPassword)
}

func TestByID(t *testing.T) {
	t.Parallel()

	pkg, ok := ByID("sonarr")
	require.True(t, ok)
	assert.Equal(t, "Sonarr", pkg.Name)

	_, ok = ByID("nonexistent")
	assert.False(t, ok)
}

// TestRegisterInstallOptions verifies that a registered provider supplies a
// package's install options at runtime, overriding any static declaration. Not
// parallel: it mutates the package-global provider registry and cleans up.
func TestRegisterInstallOptions(t *testing.T) {
	want := []model.InstallOption{{Key: "k", Label: "Label", Type: "select"}}
	RegisterInstallOptions("sonarr", func() []model.InstallOption { return want })
	t.Cleanup(func() {
		providersMu.Lock()
		delete(optionsProviders, "sonarr")
		providersMu.Unlock()
	})

	pkg, ok := ByID("sonarr")
	require.True(t, ok)
	assert.Equal(t, want, pkg.InstallOptions)
}
