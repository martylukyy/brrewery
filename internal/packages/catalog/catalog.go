package catalog

import (
	"path/filepath"

	"github.com/autobrr/brrewery/internal/packages/extravars"
	"github.com/autobrr/brrewery/internal/packages/model"
	"github.com/autobrr/brrewery/internal/packages/qbittorrent"
	"github.com/autobrr/brrewery/internal/paths"
)

func playbook(id, name string) string {
	return filepath.Join(paths.ResolveAnsibleRoot(), "playbooks", "packages", id, name+".yml")
}

func entry(id, name, desc, category, webPath string, deps []string, det *model.DetectionSpec) model.Package {
	return model.Package{
		ID:           id,
		Name:         name,
		Description:  desc,
		Category:     category,
		Icon:         iconPath(id),
		WebPath:      webPath,
		Dependencies: deps,
		Detection:    *det,
		Playbooks: model.PlaybookPaths{
			Install: playbook(id, "install"),
			Upgrade: playbook(id, "upgrade"),
			Remove:  playbook(id, "remove"),
		},
	}
}

func withInstallSecrets(pkg model.Package, specs []model.InstallSecret) model.Package {
	pkg.InstallSecrets = specs
	return pkg
}

// passwordSecret is the single install-time password prompt shared by every
// package. It collects the operator's account password — the same value is the
// Linux user password, the sudo (become) password and the brrewery dashboard
// password — and is always verified against the brrewery account before install.
func passwordSecret() model.InstallSecret {
	return model.InstallSecret{
		Key:                    extravars.BecomePassword,
		Label:                  "Password",
		Type:                   "password",
		VerifyBrreweryPassword: true,
	}
}

func qbittorrentEntry() model.Package {
	pkg := entry("qbittorrent", "qBittorrent", "BitTorrent client", "download", "/qbittorrent/",
		nil, &model.DetectionSpec{
			Binaries:         []string{"qbittorrent-nox"},
			SystemdUserUnits: []string{"qbittorrent@{user}.service"},
		})
	pkg.InstallOptions = qbittorrent.InstallOptions()
	pkg.InstallSecrets = []model.InstallSecret{passwordSecret()}
	return pkg
}

// All returns the static package catalog.
func All() []model.Package {
	return []model.Package{
		qbittorrentEntry(),
		withInstallSecrets(
			entry("autobrr", "autobrr", "Automation for torrents and *arr", "automation", "/autobrr/",
				nil, &model.DetectionSpec{
					Binaries:         []string{"autobrr"},
					SystemdUserUnits: []string{"autobrr@{user}.service"},
				}),
			[]model.InstallSecret{passwordSecret()},
		),
		entry("sonarr", "Sonarr", "TV series management", "arr", "/sonarr/",
			nil, &model.DetectionSpec{Binaries: []string{"sonarr"}, SystemdUnits: []string{"sonarr.service"}}),
		entry("radarr", "Radarr", "Movie management", "arr", "/radarr/",
			nil, &model.DetectionSpec{Binaries: []string{"radarr"}, SystemdUnits: []string{"radarr.service"}}),
		entry("prowlarr", "Prowlarr", "Indexer manager", "arr", "/prowlarr/",
			nil, &model.DetectionSpec{Binaries: []string{"prowlarr"}, SystemdUnits: []string{"prowlarr.service"}}),
		entry("lidarr", "Lidarr", "Music management", "arr", "/lidarr/",
			nil, &model.DetectionSpec{Binaries: []string{"lidarr"}, SystemdUnits: []string{"lidarr.service"}}),
		entry("sabnzbd", "SABnzbd", "Usenet downloader", "download", "/sabnzbd/",
			nil, &model.DetectionSpec{Binaries: []string{"sabnzbdplus"}, SystemdUnits: []string{"sabnzbdplus.service"}}),
		entry("deluge", "Deluge", "BitTorrent client", "download", "/deluge/",
			nil, &model.DetectionSpec{Binaries: []string{"deluged"}, SystemdUnits: []string{"deluged.service"}}),
		entry("rtorrent", "rTorrent", "BitTorrent client", "download", "",
			nil, &model.DetectionSpec{Binaries: []string{"rtorrent"}, SystemdUnits: []string{"rtorrent.service"}}),
		entry("rutorrent", "ruTorrent", "Web UI for rTorrent", "download", "/rutorrent/",
			[]string{"rtorrent"}, &model.DetectionSpec{Paths: []string{"/srv/rutorrent"}}),
		entry("jellyfin", "Jellyfin", "Media server", "media", "/jellyfin/",
			nil, &model.DetectionSpec{Binaries: []string{"jellyfin"}, SystemdUnits: []string{"jellyfin.service"}}),
		entry("plex", "Plex", "Media server", "media", "/plex/",
			nil, &model.DetectionSpec{SystemdUnits: []string{"plexmediaserver.service"}}),
		entry("filebrowser", "File Browser", "Web file manager", "tools", "/filebrowser/",
			nil, &model.DetectionSpec{Binaries: []string{"filebrowser"}, SystemdUnits: []string{"filebrowser.service"}}),
		entry("emby", "Emby", "Media server", "media", "/emby/",
			nil, &model.DetectionSpec{SystemdUnits: []string{"emby-server.service"}}),
	}
}

func ByID(id string) (model.Package, bool) {
	all := All()
	for i := range all {
		if all[i].ID == id {
			return all[i], true
		}
	}
	return model.Package{}, false
}

func DetectionSpec(id string) model.DetectionSpec {
	pkg, ok := ByID(id)
	if !ok {
		return model.DetectionSpec{}
	}
	return pkg.Detection
}
