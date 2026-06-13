package qbittorrent

import "github.com/autobrr/brrewery/internal/apps/catalog"

// qBittorrent's install options (the version picker and libtorrent branch
// picker) are derived from the vendored build manifest at runtime, so they are
// registered as a catalog options provider rather than declared statically in
// the catalog manifest. The qbittorrent package is imported by the apps
// service and HTTP handlers, so this init runs before the catalog is served.
func init() {
	catalog.RegisterInstallOptions(AppID, InstallOptions)
}
