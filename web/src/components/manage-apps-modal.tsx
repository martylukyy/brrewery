import { useMemo } from "react";
import { IconArrowUp, IconDownload, IconTrash } from "@tabler/icons-react";

import { AppIcon } from "@/components/app-icon";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { JobAction, AppStatus } from "@/lib/api";

export type ManageAppsConfirm = {
  action: JobAction;
  appIds: string[];
};

type Props = {
  apps: AppStatus[];
  onClose: () => void;
  onConfirm: (request: ManageAppsConfirm) => void;
  onTuneSysctl: () => void;
};

function sortedApps(apps: AppStatus[]): AppStatus[] {
  return [...apps].sort((a, b) => a.name.localeCompare(b.name));
}

function canInstall(app: AppStatus): boolean {
  return !app.installed && app.dependencies_satisfied;
}

function canUpgradeOrRemove(app: AppStatus): boolean {
  return app.installed;
}

export function ManageAppsModal({ apps, onClose, onConfirm, onTuneSysctl }: Props) {
  const catalog = useMemo(() => sortedApps(apps), [apps]);

  function handleAction(action: JobAction, id: string) {
    onConfirm({ action, appIds: [id] });
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="flex max-h-[90vh] flex-col gap-0 p-0 sm:max-w-2xl">
        <DialogHeader className="gap-1 border-b border-border px-5 py-4">
          <DialogTitle className="text-base">Manage server</DialogTitle>
          <DialogDescription>
            Install, upgrade, or remove an app on this host.
          </DialogDescription>
        </DialogHeader>

        <div className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto px-5 py-3">
          {catalog.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">No apps in catalog.</p>
          ) : (
            <ul className="space-y-1">
              {catalog.map((app) => {
                const installBlocked = !app.installed && !app.dependencies_satisfied;
                const installEnabled = canInstall(app);
                const modifyEnabled = canUpgradeOrRemove(app);

                return (
                  <li
                    key={app.id}
                    className="flex items-start gap-3 rounded-md px-2 py-2.5 hover:bg-accent/50"
                  >
                    <AppIcon icon={app.icon} className="size-9 self-center" />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-foreground">{app.name}</span>
                        <Badge variant={app.installed ? "secondary" : "outline"}>
                          {app.installed ? "Installed" : "Not installed"}
                        </Badge>
                      </div>
                      {app.description && (
                        <p className="mt-0.5 text-xs text-muted-foreground">{app.description}</p>
                      )}
                      {installBlocked && (
                        <p className="mt-1 text-xs text-amber-500">
                          Install required dependencies first.
                        </p>
                      )}
                    </div>
                    <div className="flex shrink-0 items-center gap-2 self-center">
                      <Button
                        variant="outline"
                        size="sm"
                        aria-label={`Install ${app.name}`}
                        disabled={!installEnabled}
                        onClick={() => handleAction("install", app.id)}
                      >
                        <IconDownload data-icon="inline-start" />
                        Install
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        aria-label={`Upgrade ${app.name}`}
                        disabled={!modifyEnabled}
                        onClick={() => handleAction("upgrade", app.id)}
                      >
                        <IconArrowUp data-icon="inline-start" />
                        Upgrade
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        aria-label={`Remove ${app.name}`}
                        disabled={!modifyEnabled}
                        onClick={() => handleAction("remove", app.id)}
                      >
                        <IconTrash data-icon="inline-start" />
                        Remove
                      </Button>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        <DialogFooter className="border-t border-border px-5 py-4 sm:justify-start">
          <Button variant="outline" onClick={onTuneSysctl}>
            Tune sysctl parameters
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
