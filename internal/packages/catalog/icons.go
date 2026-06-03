package catalog

// Icons are vendored under web/public/packages/ (official project branding).
var iconFiles = map[string]string{
	"qbittorrent": "qbittorrent.png",
	"autobrr":     "autobrr.png",
	"sonarr":      "sonarr.png",
	"radarr":      "radarr.png",
	"prowlarr":    "prowlarr.png",
	"lidarr":      "lidarr.png",
	"bazarr":      "bazarr.png",
	"sabnzbd":     "sabnzbd.png",
	"deluge":      "deluge.png",
	"rtorrent":    "rutorrent.png",
	"rutorrent":   "rutorrent.png",
	"jellyfin":    "jellyfin.png",
	"plex":        "plex.png",
	"organizr":    "organizr.png",
	"filebrowser": "filebrowser.png",
	"emby":        "emby.png",
}

func iconPath(id string) string {
	file, ok := iconFiles[id]
	if !ok {
		return ""
	}
	return "/packages/" + file
}
