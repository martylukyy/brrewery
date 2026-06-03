import { PackageIcon } from "@/components/package-icon";
import { packageAppUrl } from "@/lib/package-app";
import type { PackageStatus } from "@/lib/api";

type Props = {
  packages: PackageStatus[];
  onManageClick: () => void;
};

export function PackageNav({ packages, onManageClick }: Props) {
  const installed = packages.filter((pkg) => pkg.installed);

  return (
    <nav className="flex h-full min-h-0 flex-col bg-zinc-900/40">
      <div className="shrink-0 border-b border-zinc-800 px-3 py-3">
        <span className="font-semibold text-zinc-200">brrewery</span>
      </div>

      <ul className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto py-2">
        {installed.length === 0 && (
          <li className="px-3 py-2 text-sm text-zinc-500">No packages installed</li>
        )}
        {installed.map((pkg) => {
          const appUrl = packageAppUrl(pkg.web_path);

          if (!appUrl) {
            return (
              <li key={pkg.id}>
                <div className="flex w-full items-center gap-2 px-3 py-2 text-sm text-zinc-500">
                  <PackageIcon packageId={pkg.id} name={pkg.name} icon={pkg.icon} />
                  <span>{pkg.name}</span>
                </div>
              </li>
            );
          }

          return (
            <li key={pkg.id}>
              <a
                href={appUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="flex w-full items-center gap-2 px-3 py-2 text-sm text-zinc-300 hover:bg-zinc-800/50"
              >
                <PackageIcon packageId={pkg.id} name={pkg.name} icon={pkg.icon} />
                <span className="truncate">{pkg.name}</span>
              </a>
            </li>
          );
        })}
      </ul>

      <div className="shrink-0 border-t border-zinc-800 p-3">
        <button
          type="button"
          className="w-full rounded-md border border-zinc-700 px-3 py-2 text-sm text-zinc-300 hover:bg-zinc-800 hover:text-zinc-100"
          onClick={onManageClick}
        >
          Manage packages
        </button>
      </div>
    </nav>
  );
}
