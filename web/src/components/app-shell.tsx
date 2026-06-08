import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { Dashboard } from "@/components/dashboard";
import { InstallOptionsModal, requiredInstallOptions } from "@/components/install-options-modal";
import { InstallSecretsModal, requiredSecrets } from "@/components/install-secrets-modal";
import { ManagePackagesModal, type ManagePackagesConfirm } from "@/components/manage-packages-modal";
import { PackageJobModal } from "@/components/package-job-modal";
import { PackageNav } from "@/components/package-nav";
import { useAuth } from "@/hooks/use-auth";
import { listPackages, type JobAction } from "@/lib/api";

type ManagePhase = "select" | "secrets" | "options" | "job";

export function AppShell() {
  const { session, logout } = useAuth();
  const queryClient = useQueryClient();
  const [phase, setPhase] = useState<ManagePhase | null>(null);
  const [pendingAction, setPendingAction] = useState<JobAction>("install");
  const [pendingPackageIds, setPendingPackageIds] = useState<string[]>([]);
  const [jobQueueTotal, setJobQueueTotal] = useState(0);
  const [jobExtraVars, setJobExtraVars] = useState<Record<string, string>>({});

  const packages = useQuery({
    queryKey: ["packages"],
    queryFn: listPackages,
  });

  const packageList = packages.data?.packages ?? [];

  function beginPackageJobs({ action, packageIds }: ManagePackagesConfirm) {
    if (packageIds.length === 0) {
      return;
    }
    setPendingAction(action);
    setJobQueueTotal(packageIds.length);
    setPendingPackageIds(packageIds);
    setJobExtraVars({});
    if (action === "install" && requiredSecrets(packageList, packageIds).length > 0) {
      setPhase("secrets");
      return;
    }
    if (needsOptions(action, packageIds)) {
      setPhase("options");
      return;
    }
    setPhase("job");
  }

  function needsOptions(action: JobAction, packageIds: string[]): boolean {
    return (action === "install" || action === "upgrade") &&
      requiredInstallOptions(packageList, packageIds).length > 0;
  }

  function finishManageFlow() {
    setPhase(null);
    setPendingPackageIds([]);
    setJobQueueTotal(0);
    setJobExtraVars({});
  }

  function handleSecretsConfirm(extraVars: Record<string, string>) {
    setJobExtraVars(extraVars);
    if (needsOptions(pendingAction, pendingPackageIds)) {
      setPhase("options");
      return;
    }
    setPhase("job");
  }

  function handleOptionsConfirm(extraVars: Record<string, string>) {
    setJobExtraVars((current) => ({ ...current, ...extraVars }));
    setPhase("job");
  }

  function handleJobFinished(packageId: string) {
    setPendingPackageIds((current) => {
      const remaining = current.filter((id) => id !== packageId);
      if (remaining.length === 0) {
        setPhase(null);
        setJobQueueTotal(0);
        setJobExtraVars({});
        void queryClient.invalidateQueries({ queryKey: ["packages"] });
      }
      return remaining;
    });
  }

  const activePackageId = pendingPackageIds[0];
  const queuePosition = jobQueueTotal - pendingPackageIds.length + 1;

  return (
    <div className="flex h-screen overflow-hidden">
      <aside className="flex h-full min-h-0 w-56 shrink-0 flex-col border-r border-zinc-800 lg:w-64">
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          {packages.isLoading && (
            <p className="p-3 text-sm text-zinc-500">Loading packages…</p>
          )}
          {packages.isError && (
            <p className="p-3 text-sm text-red-400">{packages.error.message}</p>
          )}
          {packages.data && (
            <PackageNav
              packages={packageList}
              onManageClick={() => setPhase("select")}
            />
          )}
        </div>

        <div className="shrink-0 space-y-3 border-t border-zinc-800 bg-zinc-950 px-4 py-3">
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

      <main className="scrollbar-zinc min-h-0 min-w-0 flex-1 overflow-y-auto p-6">
        <Dashboard />
      </main>

      {phase === "select" && (
        <ManagePackagesModal
          packages={packageList}
          onClose={() => setPhase(null)}
          onConfirm={beginPackageJobs}
        />
      )}

      {phase === "secrets" && pendingPackageIds.length > 0 && (
        <InstallSecretsModal
          packageIds={pendingPackageIds}
          packages={packageList}
          onClose={finishManageFlow}
          onConfirm={handleSecretsConfirm}
        />
      )}

      {phase === "options" && pendingPackageIds.length > 0 && (
        <InstallOptionsModal
          packageIds={pendingPackageIds}
          packages={packageList}
          onClose={finishManageFlow}
          onConfirm={handleOptionsConfirm}
        />
      )}

      {phase === "job" && activePackageId && (
        <PackageJobModal
          key={`${pendingAction}-${activePackageId}`}
          action={pendingAction}
          packageIds={[activePackageId]}
          packages={packageList}
          extraVars={jobExtraVars}
          queuePosition={queuePosition}
          queueTotal={jobQueueTotal}
          onClose={finishManageFlow}
          onFinished={handleJobFinished}
        />
      )}
    </div>
  );
}
