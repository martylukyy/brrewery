package rtorrent

import "github.com/autobrr/brrewery/internal/apps/catalog"

// rtorrent's install options (the version picker) are derived from the vendored
// build manifest at runtime, so they are registered as a catalog options
// provider rather than declared statically in the catalog manifest. The rtorrent
// package is imported by the apps service, so this init runs before the catalog
// is served.
func init() {
	catalog.RegisterInstallOptions(AppID, InstallOptions)
}
