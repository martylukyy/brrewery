import type { AppStatus } from "@/lib/api";

export function appUrl(webPath: string | undefined): string | null {
  if (!webPath) {
    return null;
  }
  // Port form for apps served on their own port instead of through a brrewery
  // reverse proxy (e.g. Plex, like swizzin): "http://:32400/web". The host is
  // empty and filled in with the current host at runtime, keeping the given
  // scheme (Plex serves plain HTTP on :32400, avoiding a cert prompt).
  const portForm = webPath.match(/^(https?):\/\/(:\d+(?:\/.*)?)$/);
  if (portForm) {
    return `${portForm[1]}://${window.location.hostname}${portForm[2]}`;
  }
  return new URL(webPath, window.location.origin).href;
}

export function openApp(app: Pick<AppStatus, "web_path">) {
  const url = appUrl(app.web_path);
  if (!url) {
    return;
  }
  window.open(url, "_blank", "noopener,noreferrer");
}
