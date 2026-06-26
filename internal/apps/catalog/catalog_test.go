package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/brrewery/internal/apps/extravars"
	"github.com/autobrr/brrewery/internal/apps/model"
)

// TestAll_Invariants validates the contract every manifest must satisfy. It is
// intentionally data-driven: adding an app (a new manifest) requires no
// change here.
func TestAll_Invariants(t *testing.T) {
	t.Parallel()

	apps := All()
	require.NotEmpty(t, apps, "catalog should not be empty")

	ids := make(map[string]struct{}, len(apps))
	for _, app := range apps {
		assert.NotEmpty(t, app.ID, "app id must be set")
		_, dup := ids[app.ID]
		assert.False(t, dup, "duplicate app id %q", app.ID)
		ids[app.ID] = struct{}{}

		assert.NotEmpty(t, app.Name, "%s: name must be set", app.ID)
		assert.NotEmpty(t, app.Description, "%s: description must be set", app.ID)
		assert.NotEmpty(t, app.Category, "%s: category must be set", app.ID)

		assert.Contains(t, app.Playbooks.Install, app.ID, "%s: install playbook path", app.ID)
		assert.Contains(t, app.Playbooks.Upgrade, app.ID, "%s: upgrade playbook path", app.ID)
		assert.Contains(t, app.Playbooks.Remove, app.ID, "%s: remove playbook path", app.ID)

		assert.True(t, hasDetection(app.Detection), "%s: detection must declare at least one check", app.ID)

		assertSharedPasswordContract(t, app)
	}
}

// hasDetection reports whether a detection spec declares at least one check, so
// an app can never be reported as installed purely by default.
func hasDetection(d model.DetectionSpec) bool {
	return len(d.Binaries) > 0 ||
		len(d.SystemdUnits) > 0 ||
		len(d.SystemdUserUnits) > 0 ||
		len(d.Paths) > 0
}

// assertSharedPasswordContract enforces that any verified secret is the shared
// account-password prompt: same key and type, regardless of which app
// declares it.
func assertSharedPasswordContract(t *testing.T, app model.App) {
	t.Helper()
	for _, secret := range app.InstallSecrets {
		if !secret.VerifyBrreweryPassword {
			continue
		}
		assert.Equal(t, extravars.BecomePassword, secret.Key, "%s: verified secret key", app.ID)
		assert.Equal(t, "password", secret.Type, "%s: verified secret type", app.ID)
	}
}

// TestRequiresAccountPassword confirms the manifest shorthand expands to the
// shared, verified password secret.
func TestRequiresAccountPassword(t *testing.T) {
	t.Parallel()

	app, ok := ByID("qui")
	require.True(t, ok)
	require.Len(t, app.InstallSecrets, 1)
	secret := app.InstallSecrets[0]
	assert.Equal(t, extravars.BecomePassword, secret.Key)
	assert.Equal(t, "password", secret.Type)
	assert.True(t, secret.VerifyBrreweryPassword)
}

func TestByID(t *testing.T) {
	t.Parallel()

	app, ok := ByID("sonarr")
	require.True(t, ok)
	assert.Equal(t, "Sonarr", app.Name)

	_, ok = ByID("nonexistent")
	assert.False(t, ok)
}

// TestRegisterInstallOptions verifies that a registered provider supplies a
// app's install options at runtime, overriding any static declaration. Not
// parallel: it mutates the package-global provider registry and cleans up.
func TestRegisterInstallOptions(t *testing.T) {
	want := []model.InstallOption{{Key: "k", Label: "Label", Type: "select"}}
	RegisterInstallOptions("sonarr", func() []model.InstallOption { return want })
	t.Cleanup(func() {
		providersMu.Lock()
		delete(optionsProviders, "sonarr")
		providersMu.Unlock()
	})

	app, ok := ByID("sonarr")
	require.True(t, ok)
	assert.Equal(t, want, app.InstallOptions)
}
