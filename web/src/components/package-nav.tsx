import type { PackageStatus } from "@/lib/api";

type Props = {
  packages: PackageStatus[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onInstallClick: () => void;
};

export function PackageNav({ packages, selectedId, onSelect, onInstallClick }: Props) {
  const installed = packages.filter((pkg) => pkg.installed);

  return (
    <nav className="flex h-full flex-col border-r border-zinc-800 bg-zinc-900/40">
      <div className="flex items-center justify-between border-b border-zinc-800 px-3 py-3">
        <span className="text-sm font-semibold text-zinc-200">Packages</span>
        <button
          type="button"
          title="Install packages (coming soon)"
          aria-label="Install packages"
          className="flex h-8 w-8 items-center justify-center rounded-md border border-zinc-700 text-lg leading-none text-zinc-300 hover:bg-zinc-800 hover:text-zinc-100"
          onClick={onInstallClick}
        >
          +
        </button>
      </div>

      <ul className="flex-1 overflow-y-auto py-2">
        {installed.length === 0 && (
          <li className="px-3 py-2 text-sm text-zinc-500">No packages installed</li>
        )}
        {installed.map((pkg) => {
          const active = pkg.id === selectedId;
          return (
            <li key={pkg.id}>
              <button
                type="button"
                className={
                  active
                    ? "w-full px-3 py-2 text-left text-sm font-medium text-sky-300 bg-zinc-800/80"
                    : "w-full px-3 py-2 text-left text-sm text-zinc-300 hover:bg-zinc-800/50"
                }
                onClick={() => onSelect(pkg.id)}
              >
                {pkg.name}
              </button>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
