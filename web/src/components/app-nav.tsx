import { AppIcon } from "@/components/app-icon";
import { appUrl } from "@/lib/app-link";
import type { AppStatus } from "@/lib/api";

type Props = {
  apps: AppStatus[];
  onManageClick: () => void;
};

export function AppNav({ apps, onManageClick }: Props) {
  const installed = apps.filter((app) => app.installed);

  return (
    <nav className="flex h-full min-h-0 flex-col bg-zinc-900/40">
      <div className="shrink-0 border-b border-zinc-800 px-3 py-3">
        <span className="font-semibold text-zinc-200">brrewery</span>
      </div>

      <ul className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto py-2">
        {installed.length === 0 && (
          <li className="px-3 py-2 text-sm text-zinc-500">No apps installed</li>
        )}
        {installed.map((app) => {
          const url = appUrl(app.web_path);

          if (!url) {
            return (
              <li key={app.id}>
                <div className="flex w-full items-center gap-2 px-3 py-2 text-sm text-zinc-500">
                  <AppIcon icon={app.icon} className="size-5" />
                  <span>{app.name}</span>
                </div>
              </li>
            );
          }

          return (
            <li key={app.id}>
              <a
                href={url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex w-full items-center gap-2 px-3 py-2 text-sm text-zinc-300 hover:bg-zinc-800/50"
              >
                <AppIcon icon={app.icon} className="size-5" />
                <span className="truncate">{app.name}</span>
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
          Manage apps
        </button>
      </div>
    </nav>
  );
}
