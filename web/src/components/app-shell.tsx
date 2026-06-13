import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { Dashboard } from "@/components/dashboard";
import { InstallOptionsModal, requiredInstallOptions } from "@/components/install-options-modal";
import { InstallSecretsModal, requiredSecrets } from "@/components/install-secrets-modal";
import { ManageAppsModal, type ManageAppsConfirm } from "@/components/manage-apps-modal";
import { AppJobModal } from "@/components/app-job-modal";
import { AppNav } from "@/components/app-nav";
import { useAuth } from "@/hooks/use-auth";
import { listApps, type JobAction } from "@/lib/api";

type ManagePhase = "select" | "secrets" | "options" | "job";

export function AppShell() {
  const { session, logout } = useAuth();
  const queryClient = useQueryClient();
  const [phase, setPhase] = useState<ManagePhase | null>(null);
  const [pendingAction, setPendingAction] = useState<JobAction>("install");
  const [pendingAppIds, setPendingAppIds] = useState<string[]>([]);
  const [jobQueueTotal, setJobQueueTotal] = useState(0);
  const [jobExtraVars, setJobExtraVars] = useState<Record<string, string>>({});

  const apps = useQuery({
    queryKey: ["apps"],
    queryFn: listApps,
  });

  const appList = apps.data?.apps ?? [];

  function beginAppJobs({ action, appIds }: ManageAppsConfirm) {
    if (appIds.length === 0) {
      return;
    }
    setPendingAction(action);
    setJobQueueTotal(appIds.length);
    setPendingAppIds(appIds);
    setJobExtraVars({});
    if (action === "install" && requiredSecrets(appList, appIds).length > 0) {
      setPhase("secrets");
      return;
    }
    if (needsOptions(action, appIds)) {
      setPhase("options");
      return;
    }
    setPhase("job");
  }

  function needsOptions(action: JobAction, appIds: string[]): boolean {
    return (action === "install" || action === "upgrade") &&
      requiredInstallOptions(appList, appIds).length > 0;
  }

  function finishManageFlow() {
    setPhase(null);
    setPendingAppIds([]);
    setJobQueueTotal(0);
    setJobExtraVars({});
  }

  function handleSecretsConfirm(extraVars: Record<string, string>) {
    setJobExtraVars(extraVars);
    if (needsOptions(pendingAction, pendingAppIds)) {
      setPhase("options");
      return;
    }
    setPhase("job");
  }

  function handleOptionsConfirm(extraVars: Record<string, string>) {
    setJobExtraVars((current) => ({ ...current, ...extraVars }));
    setPhase("job");
  }

  function handleJobFinished(appId: string) {
    setPendingAppIds((current) => {
      const remaining = current.filter((id) => id !== appId);
      if (remaining.length === 0) {
        setPhase(null);
        setJobQueueTotal(0);
        setJobExtraVars({});
        void queryClient.invalidateQueries({ queryKey: ["apps"] });
      }
      return remaining;
    });
  }

  const activeAppId = pendingAppIds[0];
  const queuePosition = jobQueueTotal - pendingAppIds.length + 1;

  return (
    <div className="flex h-screen overflow-hidden">
      <aside className="flex h-full min-h-0 w-56 shrink-0 flex-col border-r border-zinc-800 lg:w-64">
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          {apps.isLoading && (
            <p className="p-3 text-sm text-zinc-500">Loading apps…</p>
          )}
          {apps.isError && (
            <p className="p-3 text-sm text-red-400">{apps.error.message}</p>
          )}
          {apps.data && (
            <AppNav
              apps={appList}
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
        <ManageAppsModal
          apps={appList}
          onClose={() => setPhase(null)}
          onConfirm={beginAppJobs}
        />
      )}

      {phase === "secrets" && pendingAppIds.length > 0 && (
        <InstallSecretsModal
          appIds={pendingAppIds}
          apps={appList}
          onClose={finishManageFlow}
          onConfirm={handleSecretsConfirm}
        />
      )}

      {phase === "options" && pendingAppIds.length > 0 && (
        <InstallOptionsModal
          appIds={pendingAppIds}
          apps={appList}
          onClose={finishManageFlow}
          onConfirm={handleOptionsConfirm}
        />
      )}

      {phase === "job" && activeAppId && (
        <AppJobModal
          key={`${pendingAction}-${activeAppId}`}
          action={pendingAction}
          appIds={[activeAppId]}
          apps={appList}
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
