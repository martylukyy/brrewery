import type { AppStatus } from "@/lib/api";

export function appUrl(webPath: string | undefined): string | null {
  if (!webPath) {
    return null;
  }
  // Port form for apps served on their own port instead of under a subpath of
  // the dashboard vhost (e.g. Plex, like swizzin): "https://:32443/web". The
  // host is empty and filled in with the current host at runtime, keeping the
  // given scheme. Manifests should use https: the dashboard sends HSTS with
  // includeSubDomains, so a browser upgrades an http:// link on this hostname
  // whatever the port, and the app's port has to answer TLS.
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
