import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { Dashboard } from "@/components/dashboard";
import { PackageNav } from "@/components/package-nav";
import { useAuth } from "@/hooks/use-auth";
import { listPackages } from "@/lib/api";

export function AppShell() {
  const { session, logout } = useAuth();
  const [selectedPackageId, setSelectedPackageId] = useState<string | null>(null);

  const packages = useQuery({
    queryKey: ["packages"],
    queryFn: listPackages,
  });

  const packageList = packages.data?.packages ?? [];

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-56 shrink-0 flex-col border-r border-zinc-800 lg:w-64">
        <div className="min-h-0 flex-1">
          {packages.isLoading && (
            <p className="p-3 text-sm text-zinc-500">Loading packages…</p>
          )}
          {packages.isError && (
            <p className="p-3 text-sm text-red-400">{packages.error.message}</p>
          )}
          {packages.data && (
            <PackageNav
              packages={packageList}
              selectedId={selectedPackageId}
              onSelect={setSelectedPackageId}
              onInstallClick={() => {
                // M2: open install wizard
              }}
            />
          )}
        </div>

        <div className="space-y-3 border-t border-zinc-800 bg-zinc-950 px-4 py-3">
          <div className="flex items-baseline gap-3">
            {session?.version && (
              <span className="text-xs text-zinc-500">Version {session.version}</span>
            )}
          </div>
          <button
            type="button"
            className="w-full rounded-md border border-zinc-700 px-3 py-1.5 text-sm text-zinc-300 hover:bg-zinc-900"
            onClick={() => logout.mutate()}
          >
            Log out
          </button>
        </div>
      </aside>

      <main className="min-w-0 flex-1 overflow-y-auto p-6">
        <Dashboard />
      </main>
    </div>
  );
}
