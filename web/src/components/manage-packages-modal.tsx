import { useEffect, useMemo } from "react";

import { PackageIcon } from "@/components/package-icon";
import type { JobAction, PackageStatus } from "@/lib/api";

export type ManagePackagesConfirm = {
  action: JobAction;
  packageIds: string[];
};

type Props = {
  packages: PackageStatus[];
  onClose: () => void;
  onConfirm: (request: ManagePackagesConfirm) => void;
};

function sortedPackages(packages: PackageStatus[]): PackageStatus[] {
  return [...packages].sort((a, b) => a.name.localeCompare(b.name));
}

function canInstall(pkg: PackageStatus): boolean {
  return !pkg.installed && pkg.dependencies_satisfied;
}

function canUpgradeOrRemove(pkg: PackageStatus): boolean {
  return pkg.installed;
}

export function ManagePackagesModal({ packages, onClose, onConfirm }: Props) {
  const catalog = useMemo(() => sortedPackages(packages), [packages]);

  function handleAction(action: JobAction, id: string) {
    onConfirm({ action, packageIds: [id] });
  }

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onClose();
      }
    }

    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60" aria-hidden="true" />

      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="manage-packages-title"
        className="relative z-10 flex h-full max-h-[90%] w-full max-w-[90%] flex-col rounded-lg border border-zinc-700 bg-zinc-900 shadow-xl"
      >
        <div className="flex items-start justify-between gap-4 border-b border-zinc-800 px-5 py-4">
          <div>
            <h2 id="manage-packages-title" className="text-lg font-semibold text-zinc-100">
              Manage packages
            </h2>
            <p className="mt-1 text-sm text-zinc-400">
              Install, upgrade, or remove a package on this host.
            </p>
          </div>
          <button
            type="button"
            className="-mr-1 -mt-1 shrink-0 rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-zinc-100"
            aria-label="Close manage packages dialog"
            onClick={onClose}
          >
            <svg
              viewBox="0 0 24 24"
              className="size-8"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              aria-hidden="true"
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 6l12 12M18 6L6 18" />
            </svg>
          </button>
        </div>

        <div className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto px-5 py-3">
          {catalog.length === 0 ? (
            <p className="py-6 text-center text-sm text-zinc-500">No packages in catalog.</p>
          ) : (
            <ul className="space-y-1">
              {catalog.map((pkg) => {
                const installBlocked = !pkg.installed && !pkg.dependencies_satisfied;
                const installEnabled = canInstall(pkg);
                const modifyEnabled = canUpgradeOrRemove(pkg);

                return (
                  <li
                    key={pkg.id}
                    className="flex items-start gap-3 rounded-md px-2 py-2.5 hover:bg-zinc-800/50"
                  >
                    <PackageIcon icon={pkg.icon} className="size-9 self-center" />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-zinc-100">{pkg.name}</span>
                        <span
                          className={
                            pkg.installed
                              ? "rounded-full bg-emerald-900/40 px-2 py-0.5 text-[10px] text-emerald-300"
                              : "rounded-full bg-zinc-800 px-2 py-0.5 text-[10px] text-zinc-400"
                          }
                        >
                          {pkg.installed ? "Installed" : "Not installed"}
                        </span>
                      </div>
                      {pkg.description && (
                        <p className="mt-0.5 text-xs text-zinc-500">{pkg.description}</p>
                      )}
                      {installBlocked && (
                        <p className="mt-1 text-xs text-amber-400/90">
                          Install required dependencies first.
                        </p>
                      )}
                    </div>
                    <div className="flex shrink-0 items-center gap-2 self-center">
                      <button
                        type="button"
                        className="rounded-md border border-zinc-700 px-3 py-1.5 text-sm text-zinc-300 hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-50"
                        aria-label={`Install ${pkg.name}`}
                        disabled={!installEnabled}
                        onClick={() => handleAction("install", pkg.id)}
                      >
                        Install
                      </button>
                      <button
                        type="button"
                        className="rounded-md border border-zinc-700 px-3 py-1.5 text-sm text-zinc-300 hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-50"
                        aria-label={`Upgrade ${pkg.name}`}
                        disabled={!modifyEnabled}
                        onClick={() => handleAction("upgrade", pkg.id)}
                      >
                        Upgrade
                      </button>
                      <button
                        type="button"
                        className="rounded-md border border-red-900/60 px-3 py-1.5 text-sm text-red-300 hover:bg-red-950/40 disabled:cursor-not-allowed disabled:opacity-50"
                        aria-label={`Remove ${pkg.name}`}
                        disabled={!modifyEnabled}
                        onClick={() => handleAction("remove", pkg.id)}
                      >
                        Remove
                      </button>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        <div className="flex justify-end border-t border-zinc-800 px-5 py-4">
          <button
            type="button"
            className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:bg-zinc-800"
            onClick={onClose}
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
