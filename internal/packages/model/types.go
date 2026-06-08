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

type Package struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Category       string          `json:"category"`
	Icon           string          `json:"icon,omitempty"`
	WebPath        string          `json:"web_path,omitempty"`
	InstallSecrets []InstallSecret `json:"install_secrets,omitempty"`
	InstallOptions []InstallOption `json:"install_options,omitempty"`
	Dependencies   []string        `json:"dependencies,omitempty"`
	Detection      DetectionSpec   `json:"detection"`
	Playbooks      PlaybookPaths   `json:"playbooks"`
}

type PackageStatus struct {
	Package
	Installed             bool `json:"installed"`
	DependenciesSatisfied bool `json:"dependencies_satisfied"`
}
