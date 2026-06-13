import type { AppStatus } from "@/lib/api";

type Props = {
  apps: AppStatus[];
};

export function AppsTable({ apps }: Props) {
  return (
    <div className="overflow-x-auto rounded-lg border border-zinc-800">
      <table className="min-w-full text-left text-sm">
        <thead className="bg-zinc-900 text-zinc-400">
          <tr>
            <th className="px-4 py-3 font-medium">Name</th>
            <th className="px-4 py-3 font-medium">Category</th>
            <th className="px-4 py-3 font-medium">Status</th>
            <th className="px-4 py-3 font-medium">Description</th>
          </tr>
        </thead>
        <tbody>
          {apps.map((app) => (
            <tr key={app.id} className="border-t border-zinc-800">
              <td className="px-4 py-3 font-medium text-zinc-100">{app.name}</td>
              <td className="px-4 py-3 text-zinc-400">{app.category}</td>
              <td className="px-4 py-3">
                <span
                  className={
                    app.installed
                      ? "rounded-full bg-emerald-900/50 px-2 py-0.5 text-emerald-300"
                      : "rounded-full bg-zinc-800 px-2 py-0.5 text-zinc-400"
                  }
                >
                  {app.installed ? "Installed" : "Not installed"}
                </span>
              </td>
              <td className="px-4 py-3 text-zinc-400">{app.description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
