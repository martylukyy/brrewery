import { useEffect, useMemo, useState } from "react";

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

function canInstall(packages: PackageStatus[], ids: string[]): boolean {
  if (ids.length === 0) {
    return false;
  }
  return ids.every((id) => {
    const pkg = packages.find((entry) => entry.id === id);
    return pkg != null && !pkg.installed && pkg.dependencies_satisfied;
  });
}

function canUpgradeOrRemove(packages: PackageStatus[], ids: string[]): boolean {
  if (ids.length === 0) {
    return false;
  }
  return ids.every((id) => packages.find((entry) => entry.id === id)?.installed);
}

export function ManagePackagesModal({ packages, onClose, onConfirm }: Props) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const catalog = useMemo(() => sortedPackages(packages), [packages]);

  const selected = [...selectedIds];
  const installEnabled = canInstall(packages, selected);
  const upgradeEnabled = canUpgradeOrRemove(packages, selected);
  const removeEnabled = canUpgradeOrRemove(packages, selected);

  function togglePackage(id: string) {
    setSelectedIds((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  function handleAction(action: JobAction) {
    onConfirm({ action, packageIds: selected });
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
      <button
        type="button"
        className="absolute inset-0 bg-black/60"
        aria-label="Close manage packages dialog"
        onClick={onClose}
      />

      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="manage-packages-title"
        className="relative z-10 flex max-h-[min(32rem,85vh)] w-full max-w-2xl flex-col rounded-lg border border-zinc-700 bg-zinc-900 shadow-xl"
      >
        <div className="border-b border-zinc-800 px-5 py-4">
          <h2 id="manage-packages-title" className="text-lg font-semibold text-zinc-100">
            Manage packages
          </h2>
          <p className="mt-1 text-sm text-zinc-400">
            Select packages to install, upgrade, or remove on this host.
          </p>
        </div>

        <div className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto px-5 py-3">
          {catalog.length === 0 ? (
            <p className="py-6 text-center text-sm text-zinc-500">No packages in catalog.</p>
          ) : (
            <ul className="space-y-1">
              {catalog.map((pkg) => {
                const checked = selectedIds.has(pkg.id);
                const installBlocked = !pkg.installed && !pkg.dependencies_satisfied;

                return (
                  <li key={pkg.id}>
                    <label
                      className={
                        installBlocked
                          ? "flex cursor-pointer items-start gap-3 rounded-md px-2 py-2 hover:bg-zinc-800/50"
                          : "flex cursor-pointer items-start gap-3 rounded-md px-2 py-2 hover:bg-zinc-800/50"
                      }
                    >
                      <input
                        type="checkbox"
                        className="mt-0.5 rounded border-zinc-600 bg-zinc-950 text-amber-600 focus:ring-amber-500/50"
                        checked={checked}
                        onChange={() => togglePackage(pkg.id)}
                      />
                      <span className="min-w-0 flex-1">
                        <span className="flex items-center gap-2">
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
                        </span>
                        {pkg.description && (
                          <span className="mt-0.5 block text-xs text-zinc-500">{pkg.description}</span>
                        )}
                        {installBlocked && (
                          <span className="mt-1 block text-xs text-amber-400/90">
                            Install required dependencies first.
                          </span>
                        )}
                      </span>
                      <span className="shrink-0 rounded-full bg-zinc-800 px-2 py-0.5 text-xs text-zinc-400">
                        {pkg.category}
                      </span>
                    </label>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-zinc-800 px-5 py-4">
          <button
            type="button"
            className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:bg-zinc-800"
            onClick={onClose}
          >
            Cancel
          </button>
          <div className="flex flex-wrap justify-end gap-2">
            <button
              type="button"
              className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-50"
              disabled={!installEnabled}
              onClick={() => handleAction("install")}
            >
              Install
            </button>
            <button
              type="button"
              className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-50"
              disabled={!upgradeEnabled}
              onClick={() => handleAction("upgrade")}
            >
              Upgrade
            </button>
            <button
              type="button"
              className="rounded-md border border-red-900/60 px-4 py-2 text-sm text-red-300 hover:bg-red-950/40 disabled:cursor-not-allowed disabled:opacity-50"
              disabled={!removeEnabled}
              onClick={() => handleAction("remove")}
            >
              Remove
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export { canInstall, canUpgradeOrRemove };
