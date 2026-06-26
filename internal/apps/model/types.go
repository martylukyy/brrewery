package model

type DetectionSpec struct {
	Binaries         []string `json:"binaries,omitempty" yaml:"binaries,omitempty"`
	SystemdUnits     []string `json:"systemd_units,omitempty" yaml:"systemd_units,omitempty"`
	SystemdUserUnits []string `json:"systemd_user_units,omitempty" yaml:"systemd_user_units,omitempty"`
	Paths            []string `json:"paths,omitempty" yaml:"paths,omitempty"`
	DependsOn        []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
}

type PlaybookPaths struct {
	Install string `json:"install"`
	Upgrade string `json:"upgrade"`
	Remove  string `json:"remove"`
}

type InstallSecret struct {
	Key                    string `json:"key" yaml:"key"`
	Label                  string `json:"label" yaml:"label"`
	Type                   string `json:"type" yaml:"type"`
	VerifyBrreweryPassword bool   `json:"verify_brrewery_password,omitempty" yaml:"verify_brrewery_password,omitempty"`
	// DisablePasswordManager hints browsers and extensions not to autofill or save the value.
	DisablePasswordManager bool `json:"disable_password_manager,omitempty" yaml:"disable_password_manager,omitempty"`
}

// InstallOptionChoice is a single selectable value for an InstallOption.
type InstallOptionChoice struct {
	Value string `json:"value" yaml:"value"`
	Label string `json:"label" yaml:"label"`
}

// InstallOptionWhen gates an option so it is only shown when another option's
// value is one of the listed values.
type InstallOptionWhen struct {
	Key   string   `json:"key" yaml:"key"`
	OneOf []string `json:"one_of,omitempty" yaml:"one_of,omitempty"`
}

// InstallOption is a user-selected build/install parameter passed to Ansible as
// an extra var (e.g. the qBittorrent version or libtorrent branch).
type InstallOption struct {
	Key     string                `json:"key" yaml:"key"`
	Label   string                `json:"label" yaml:"label"`
	Type    string                `json:"type" yaml:"type"`
	Choices []InstallOptionChoice `json:"choices,omitempty" yaml:"choices,omitempty"`
	When    *InstallOptionWhen    `json:"when,omitempty" yaml:"when,omitempty"`
}

type App struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Category       string          `json:"category"`
	WebPath        string          `json:"web_path,omitempty"`
	InstallSecrets []InstallSecret `json:"install_secrets,omitempty"`
	InstallOptions []InstallOption `json:"install_options,omitempty"`
	Dependencies   []string        `json:"dependencies,omitempty"`
	Detection      DetectionSpec   `json:"detection"`
	Playbooks      PlaybookPaths   `json:"playbooks"`
}

// ServiceStatus reports the live systemd state of an installed app's
// controllable unit(s). It is decoupled from AppStatus.Installed (which tracks
// persistent artifacts) so an app stays listed while its service is stopped or
// disabled. The dashboard renders a toggle from Active && Enabled.
type ServiceStatus struct {
	// Units are the resolved unit names this app controls ({user} expanded).
	Units []string `json:"units"`
	// Active is true when every unit is running (systemctl is-active).
	Active bool `json:"active"`
	// Enabled is true when every unit is enabled (systemctl is-enabled).
	Enabled bool `json:"enabled"`
	// Failing is true when any unit is unhealthy — failed outright or stuck in a
	// restart loop (crash-looping). It is independent of Active: a crash-looping
	// unit never reaches "running" (Active=false) yet must be flagged so the
	// dashboard can draw a red backdrop behind the switch.
	Failing bool `json:"failing"`
}

type AppStatus struct {
	App
	Installed             bool `json:"installed"`
	DependenciesSatisfied bool `json:"dependencies_satisfied"`
	// Service is present only for installed apps that expose a controllable
	// systemd unit; apps without a service (e.g. ruTorrent) leave it nil.
	Service *ServiceStatus `json:"service,omitempty"`
}
