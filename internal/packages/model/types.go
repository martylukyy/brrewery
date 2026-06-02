package model

type DetectionSpec struct {
	Binaries     []string `json:"binaries,omitempty"`
	SystemdUnits []string `json:"systemd_units,omitempty"`
	Paths        []string `json:"paths,omitempty"`
	DependsOn    []string `json:"depends_on,omitempty"`
}

type PlaybookPaths struct {
	Install string `json:"install"`
	Upgrade string `json:"upgrade"`
	Remove  string `json:"remove"`
}

type Package struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Category     string        `json:"category"`
	Dependencies []string      `json:"dependencies,omitempty"`
	Detection    DetectionSpec `json:"detection"`
	Playbooks    PlaybookPaths `json:"playbooks"`
}

type PackageStatus struct {
	Package
	Installed             bool `json:"installed"`
	DependenciesSatisfied bool `json:"dependencies_satisfied"`
}
