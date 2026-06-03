#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="${ROOT}/web/public/packages"

mkdir -p "${DEST}"

curl -fsSL -o "${DEST}/autobrr.png" "https://raw.githubusercontent.com/autobrr/autobrr/develop/.github/images/logo.png"
curl -fsSL -o "${DEST}/sonarr.png" "https://sonarr.tv/img/logo.png"
curl -fsSL -o "${DEST}/radarr.png" "https://raw.githubusercontent.com/Radarr/Radarr/develop/Logo/256.png"
curl -fsSL -o "${DEST}/prowlarr.png" "https://raw.githubusercontent.com/Prowlarr/Prowlarr/develop/Logo/256.png"
curl -fsSL -o "${DEST}/lidarr.png" "https://raw.githubusercontent.com/Lidarr/Lidarr/develop/Logo/256.png"
curl -fsSL -o "${DEST}/deluge.png" "https://raw.githubusercontent.com/deluge-torrent/deluge/develop/deluge/ui/data/pixmaps/deluge.png"
curl -fsSL -o "${DEST}/rutorrent.png" "https://raw.githubusercontent.com/Novik/ruTorrent/master/images/logo.png"
curl -fsSL -o "${DEST}/filebrowser.png" "https://raw.githubusercontent.com/filebrowser/logo/master/icon.png"

# PNG-only sources where upstream ships ICO or blocks direct favicon fetches (dashboard-icons artwork).
curl -fsSL -o "${DEST}/qbittorrent.png" "https://raw.githubusercontent.com/walkxcode/dashboard-icons/main/png/qbittorrent.png"
curl -fsSL -o "${DEST}/sabnzbd.png" "https://raw.githubusercontent.com/walkxcode/dashboard-icons/main/png/sabnzbd.png"
curl -fsSL -o "${DEST}/jellyfin.png" "https://raw.githubusercontent.com/walkxcode/dashboard-icons/main/png/jellyfin.png"
curl -fsSL -o "${DEST}/emby.png" "https://raw.githubusercontent.com/walkxcode/dashboard-icons/main/png/emby.png"
curl -fsSL -o "${DEST}/plex.png" "https://raw.githubusercontent.com/walkxcode/dashboard-icons/main/png/plex.png"

echo "Package icons updated in ${DEST}"
