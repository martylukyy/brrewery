package catalog

import (
	"path/filepath"

	"github.com/autobrr/brrewery/internal/packages/model"
	"github.com/autobrr/brrewery/internal/paths"
)

func playbook(id, name string) string {
	return filepath.Join(paths.AnsibleRoot, "playbooks", "packages", id, name+".yml")
}

func entry(id, name, desc, category string, deps []string, det *model.DetectionSpec) model.Package {
	return model.Package{
		ID:           id,
		Name:         name,
		Description:  desc,
		Category:     category,
		Dependencies: deps,
		Detection:    *det,
		Playbooks: model.PlaybookPaths{
			Install: playbook(id, "install"),
			Upgrade: playbook(id, "upgrade"),
			Remove:  playbook(id, "remove"),
		},
	}
}

// All returns the static package catalog.
func All() []model.Package {
	return []model.Package{
		entry("qbittorrent", "qBittorrent", "BitTorrent client", "download",
			nil, &model.DetectionSpec{Binaries: []string{"qbittorrent-nox"}, SystemdUnits: []string{"qbittorrent.service"}}),
		entry("autobrr", "autobrr", "Automation for torrents and *arr", "automation",
			nil, &model.DetectionSpec{Binaries: []string{"autobrr"}, SystemdUnits: []string{"autobrr.service"}}),
		entry("sonarr", "Sonarr", "TV series management", "arr",
			nil, &model.DetectionSpec{Binaries: []string{"sonarr"}, SystemdUnits: []string{"sonarr.service"}}),
		entry("radarr", "Radarr", "Movie management", "arr",
			nil, &model.DetectionSpec{Binaries: []string{"radarr"}, SystemdUnits: []string{"radarr.service"}}),
		entry("prowlarr", "Prowlarr", "Indexer manager", "arr",
			nil, &model.DetectionSpec{Binaries: []string{"prowlarr"}, SystemdUnits: []string{"prowlarr.service"}}),
		entry("lidarr", "Lidarr", "Music management", "arr",
			nil, &model.DetectionSpec{Binaries: []string{"lidarr"}, SystemdUnits: []string{"lidarr.service"}}),
		entry("bazarr", "Bazarr", "Subtitle management", "arr",
			nil, &model.DetectionSpec{Binaries: []string{"bazarr"}, SystemdUnits: []string{"bazarr.service"}}),
		entry("sabnzbd", "SABnzbd", "Usenet downloader", "download",
			nil, &model.DetectionSpec{Binaries: []string{"sabnzbdplus"}, SystemdUnits: []string{"sabnzbdplus.service"}}),
		entry("deluge", "Deluge", "BitTorrent client", "download",
			nil, &model.DetectionSpec{Binaries: []string{"deluged"}, SystemdUnits: []string{"deluged.service"}}),
		entry("rtorrent", "rTorrent", "BitTorrent client", "download",
			nil, &model.DetectionSpec{Binaries: []string{"rtorrent"}, SystemdUnits: []string{"rtorrent.service"}}),
		entry("rutorrent", "ruTorrent", "Web UI for rTorrent", "download",
			[]string{"rtorrent"}, &model.DetectionSpec{Paths: []string{"/srv/rutorrent"}}),
		entry("jellyfin", "Jellyfin", "Media server", "media",
			nil, &model.DetectionSpec{Binaries: []string{"jellyfin"}, SystemdUnits: []string{"jellyfin.service"}}),
		entry("plex", "Plex", "Media server", "media",
			nil, &model.DetectionSpec{SystemdUnits: []string{"plexmediaserver.service"}}),
		entry("organizr", "Organizr", "HTPC dashboard", "dashboard",
			nil, &model.DetectionSpec{Paths: []string{"/srv/organizr"}}),
		entry("filebrowser", "File Browser", "Web file manager", "tools",
			nil, &model.DetectionSpec{Binaries: []string{"filebrowser"}, SystemdUnits: []string{"filebrowser.service"}}),
		entry("emby", "Emby", "Media server", "media",
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
