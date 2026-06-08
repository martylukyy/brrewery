import { useState } from "react";

type Props = {
  packageId: string;
  name: string;
  icon?: string;
};

const FALLBACK_COLORS: Record<string, string> = {
  autobrr: "#2563eb",
  sonarr: "#35c5f4",
  radarr: "#ffc230",
  prowlarr: "#f59e0b",
  lidarr: "#34d399",
  qbittorrent: "#2563eb",
  qui: "#3b82f6",
  sabnzbd: "#f97316",
  deluge: "#0891b2",
  rtorrent: "#64748b",
  rutorrent: "#84cc16",
  jellyfin: "#a855f7",
  plex: "#eab308",
  filebrowser: "#0ea5e9",
  emby: "#52b54b",
};

export function PackageIcon({ packageId, name, icon }: Props) {
  const label = name.trim().charAt(0).toUpperCase() || "?";
  const color = FALLBACK_COLORS[packageId] ?? "#52525b";
  const [imageFailed, setImageFailed] = useState(false);

  if (!icon || imageFailed) {
    return (
      <span
        aria-hidden="true"
        className="flex h-5 w-5 shrink-0 items-center justify-center rounded-md text-[10px] font-semibold text-white"
        style={{ backgroundColor: color }}
      >
        {label}
      </span>
    );
  }

  return (
    <img
      src={icon}
      alt=""
      className="h-5 w-5 shrink-0 rounded-md object-cover"
      onError={() => setImageFailed(true)}
    />
  );
}
