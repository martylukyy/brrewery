import type { InstallOption, AppStatus } from "@/lib/api";

/**
 * requiredInstallOptions returns the install options of the first selected
 * app that declares any. Only qBittorrent does today.
 */
export function requiredInstallOptions(apps: AppStatus[], appIds: string[]): InstallOption[] {
  for (const id of appIds) {
    const app = apps.find((entry) => entry.id === id);
    if (app?.install_options?.length) {
      return app.install_options;
    }
  }
  return [];
}
