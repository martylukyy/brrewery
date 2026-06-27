// App icons, keyed by catalog app id. Each SVG is imported as a URL so Vite
// emits it as a separate hashed asset (served like the font woff2 files, see
// vite.config assetsInlineLimit) rather than inlining it into the JS chunk.
// Icons ship with the frontend bundle, so there is no served /apps/ asset
// folder and the backend hands out no icon paths.
//
// The registry is built at build time from the contents of the app-icons
// folder: dropping `<id>.svg` into ../assets/app-icons registers that id
// automatically, with no edit to this file required. The id is the file name
// without its extension.
const icons = import.meta.glob<string>("@/assets/app-icons/*.svg", {
  eager: true,
  query: "?url",
  import: "default",
});

// Aliases let one id reuse another's logo. ruTorrent ships no logo of its own
// and reuses rTorrent's. Keyed by the new id, valued by the id whose icon it
// borrows — only needed for these shared-logo cases.
const ICON_ALIASES: Record<string, string> = {
  rutorrent: "rtorrent",
};

// Maps a catalog app id to its icon asset URL.
export const APP_ICONS: Record<string, string> = {};

for (const [path, url] of Object.entries(icons)) {
  // ".../app-icons/qbittorrent.svg" -> "qbittorrent"
  const id = path.split("/").pop()!.replace(/\.svg$/, "");
  APP_ICONS[id] = url;
}

for (const [alias, source] of Object.entries(ICON_ALIASES)) {
  if (APP_ICONS[source]) {
    APP_ICONS[alias] = APP_ICONS[source];
  }
}
