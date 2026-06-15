import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";

import { Dashboard } from "@/components/dashboard";
import { InstallOptionsModal } from "@/components/install-options-modal";
import { InstallSecretsModal } from "@/components/install-secrets-modal";
import { requiredInstallOptions } from "@/lib/install-options";
import { requiredSecrets } from "@/lib/install-secrets";
import { ManageAppsModal, type ManageAppsConfirm } from "@/components/manage-apps-modal";
import { SysctlModal } from "@/components/sysctl-modal";
import { AppJobModal } from "@/components/app-job-modal";
import { AppSidebar } from "@/components/app-sidebar";
import { ServiceToggleModal } from "@/components/service-toggle-modal";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { useAuth } from "@/hooks/use-auth";
import {
  ApiError,
  listApps,
  setAppService,
  verifyPassword,
  type AppStatus,
  type JobAction,
} from "@/lib/api";

type ManagePhase = "select" | "secrets" | "options" | "job" | "sysctl";

type ServiceToggleRequest = {
  app: AppStatus;
  enabled: boolean;
  password: string;
};

export function AppShell() {
  const { session, username, logout } = useAuth();
  const queryClient = useQueryClient();
  const [phase, setPhase] = useState<ManagePhase | null>(null);
  const [pendingAction, setPendingAction] = useState<JobAction>("install");
  const [pendingAppIds, setPendingAppIds] = useState<string[]>([]);
  const [jobQueueTotal, setJobQueueTotal] = useState(0);
  const [jobExtraVars, setJobExtraVars] = useState<Record<string, string>>({});
  const [serviceToggle, setServiceToggle] = useState<{ app: AppStatus; enabled: boolean } | null>(
    null,
  );

  const apps = useQuery({
    queryKey: ["apps"],
    queryFn: listApps,
  });

  const appList = apps.data?.apps ?? [];

  // Toggling a service runs in the background after the password modal closes,
  // so the work outlives the modal and a spinner can sit where the switch was.
  // The password is verified against the credential endpoint first: a wrong
  // password there is a 401 that does not sign the user out, unlike a 401 from
  // the service endpoint which the global handler treats as an expired session.
  const serviceMutation = useMutation({
    mutationFn: async ({ app, enabled, password }: ServiceToggleRequest) => {
      await verifyPassword(password);
      await setAppService(app.id, enabled, password);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["apps"] });
    },
    onError: (error, { app, enabled }) => {
      const verb = enabled ? "start" : "stop";
      const reason =
        error instanceof ApiError && error.status === 401
          ? "Incorrect password."
          : error instanceof Error
            ? error.message
            : "Please try again.";
      toast.error(`Could not ${verb} ${app.name}. ${reason}`);
    },
  });
  // While a toggle is in flight, the targeted app's switch is replaced by a
  // spinner (see AppSidebar).
  const pendingServiceAppId = serviceMutation.isPending
    ? serviceMutation.variables?.app.id
    : undefined;

  function beginAppJobs({ action, appIds }: ManageAppsConfirm) {
    if (appIds.length === 0) {
      return;
    }
    setPendingAction(action);
    setJobQueueTotal(appIds.length);
    setPendingAppIds(appIds);
    setJobExtraVars({});
    if (requiredSecrets(appList, appIds, action).length > 0) {
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
    <SidebarProvider className="h-svh overflow-hidden">
      <AppSidebar
        apps={appList}
        isLoading={apps.isLoading}
        isError={apps.isError}
        errorMessage={apps.error?.message}
        version={session?.version}
        user={username}
        onManageClick={() => setPhase("select")}
        onLogout={() => logout.mutate()}
        onToggleService={(app, enabled) => setServiceToggle({ app, enabled })}
        pendingServiceAppId={pendingServiceAppId}
      />

      <SidebarInset className="min-h-0">
        {/*
          Below md the sidebar is an off-canvas sheet whose toggle (now in the
          sidebar header) is hidden while it's closed, so keep a minimal trigger
          here to open it. On md+ the toggle lives in the sidebar header and this
          row collapses away — no dashboard header on desktop.
        */}
        <header className="flex h-12 shrink-0 items-center border-b border-border px-2 md:hidden">
          <SidebarTrigger />
        </header>
        <div className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto p-6">
          <Dashboard />
        </div>
      </SidebarInset>

      {phase === "select" && (
        <ManageAppsModal
          apps={appList}
          onClose={() => setPhase(null)}
          onConfirm={beginAppJobs}
          onTuneSysctl={() => setPhase("sysctl")}
        />
      )}

      {phase === "sysctl" && <SysctlModal onClose={() => setPhase(null)} />}

      {phase === "secrets" && pendingAppIds.length > 0 && (
        <InstallSecretsModal
          action={pendingAction}
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

      {serviceToggle && (
        <ServiceToggleModal
          app={serviceToggle.app}
          enabled={serviceToggle.enabled}
          onClose={() => setServiceToggle(null)}
          onConfirm={(password) => {
            const { app, enabled } = serviceToggle;
            setServiceToggle(null);
            serviceMutation.mutate({ app, enabled, password });
          }}
        />
      )}
    </SidebarProvider>
  );
}
