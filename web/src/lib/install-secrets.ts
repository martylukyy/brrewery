import type { InstallSecret, AppStatus } from "@/lib/api";

/** Deduplicated install secrets declared by the selected apps, in order. */
export function requiredSecrets(apps: AppStatus[], appIds: string[]): InstallSecret[] {
  const seen = new Set<string>();
  const out: InstallSecret[] = [];

  for (const id of appIds) {
    const app = apps.find((entry) => entry.id === id);
    for (const secret of app?.install_secrets ?? []) {
      if (seen.has(secret.key)) {
        continue;
      }
      seen.add(secret.key);
      out.push(secret);
    }
  }

  return out;
}
