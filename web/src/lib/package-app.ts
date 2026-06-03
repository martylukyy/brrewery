import type { PackageStatus } from "@/lib/api";

export function packageAppUrl(webPath: string | undefined): string | null {
  if (!webPath) {
    return null;
  }
  return new URL(webPath, window.location.origin).href;
}

export function openPackageApp(pkg: Pick<PackageStatus, "web_path">) {
  const url = packageAppUrl(pkg.web_path);
  if (!url) {
    return;
  }
  window.open(url, "_blank", "noopener,noreferrer");
}
