import type { InstallSecret, AppStatus, AppJobAction } from "@/lib/api";

/**
 * Deduplicated install secrets prompted for the given action, in order.
 *
 * Install collects every secret an app declares (account password plus any
 * app-specific provisioning credentials). Upgrade and remove only re-run
 * privileged playbooks, so they ask for the account password alone — the
 * app-specific credentials are install-time only.
 */
export function requiredSecrets(
  apps: AppStatus[],
  appIds: string[],
  action: AppJobAction = "install",
): InstallSecret[] {
  const seen = new Set<string>();
  const out: InstallSecret[] = [];

  for (const id of appIds) {
    const app = apps.find((entry) => entry.id === id);
    for (const secret of app?.install_secrets ?? []) {
      if (action !== "install" && !secret.verify_brrewery_password) {
        continue;
      }
      if (seen.has(secret.key)) {
        continue;
      }
      seen.add(secret.key);
      out.push(secret);
    }
  }

  return out;
}
