package model

type DetectionSpec struct {
	Binaries         []string `json:"binaries,omitempty"`
	SystemdUnits     []string `json:"systemd_units,omitempty"`
	SystemdUserUnits []string `json:"systemd_user_units,omitempty"`
	Paths            []string `json:"paths,omitempty"`
	DependsOn        []string `json:"depends_on,omitempty"`
}

type PlaybookPaths struct {
	Install string `json:"install"`
	Upgrade string `json:"upgrade"`
	Remove  string `json:"remove"`
}

type InstallSecret struct {
	Key                    string `json:"key"`
	Label                  string `json:"label"`
	Type                   string `json:"type"`
	VerifyBrreweryPassword bool   `json:"verify_brrewery_password,omitempty"`
}

type Package struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Category       string        `json:"category"`
	Icon           string        `json:"icon,omitempty"`
	WebPath        string        `json:"web_path,omitempty"`
	InstallSecrets []InstallSecret `json:"install_secrets,omitempty"`
	Dependencies   []string      `json:"dependencies,omitempty"`
	Detection      DetectionSpec `json:"detection"`
	Playbooks      PlaybookPaths `json:"playbooks"`
}

type PackageStatus struct {
	Package
	Installed             bool `json:"installed"`
	DependenciesSatisfied bool `json:"dependencies_satisfied"`
}
