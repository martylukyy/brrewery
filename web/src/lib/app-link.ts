import type { AppStatus } from "@/lib/api";

export function appUrl(webPath: string | undefined): string | null {
  if (!webPath) {
    return null;
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
