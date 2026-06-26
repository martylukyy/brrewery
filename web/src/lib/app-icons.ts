// App icons, keyed by catalog app id. Each SVG is imported as a URL so Vite
// emits it as a separate hashed asset (served like the font woff2 files, see
// vite.config assetsInlineLimit) rather than inlining it into the JS chunk.
// Icons ship with the frontend bundle, so there is no served /apps/ asset
// folder and the backend hands out no icon paths.
import autobrr from "@/assets/app-icons/autobrr.svg";
import deluge from "@/assets/app-icons/deluge.svg";
import jellyfin from "@/assets/app-icons/jellyfin.svg";
import lidarr from "@/assets/app-icons/lidarr.svg";
import plex from "@/assets/app-icons/plex.svg";
import prowlarr from "@/assets/app-icons/prowlarr.svg";
import qbittorrent from "@/assets/app-icons/qbittorrent.svg";
import qui from "@/assets/app-icons/qui.svg";
import radarr from "@/assets/app-icons/radarr.svg";
import rtorrent from "@/assets/app-icons/rtorrent.svg";
import sabnzbd from "@/assets/app-icons/sabnzbd.svg";
import sonarr from "@/assets/app-icons/sonarr.svg";
import transmission from "@/assets/app-icons/transmission.svg";

// Maps a catalog app id to its icon asset URL. ruTorrent ships no logo of its
// own and reuses rTorrent's.
export const APP_ICONS: Record<string, string> = {
  autobrr,
  deluge,
  jellyfin,
  lidarr,
  plex,
  prowlarr,
  qbittorrent,
  qui,
  radarr,
  rtorrent,
  rutorrent: rtorrent,
  sabnzbd,
  sonarr,
  transmission,
};
