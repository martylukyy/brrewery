import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useId, useMemo, useRef, useState } from "react";
import { IconAlertTriangle, IconCheck, IconCopy, IconLoader2 } from "@tabler/icons-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  ApiError,
  getJob,
  getJobLogs,
  startAppJob,
  type JobAction,
  type JobStatus,
  type AppStatus,
} from "@/lib/api";

type Props = {
  action: JobAction;
  appIds: string[];
  apps: AppStatus[];
  extraVars?: Record<string, string>;
  queuePosition: number;
  queueTotal: number;
  onClose: () => void;
  onFinished: (appId: string) => void;
};

const TERMINAL: JobStatus[] = ["succeeded", "failed"];

const ACTION_LABELS: Record<JobAction, { title: string; running: string; failedStart: string }> = {
  install: {
    title: "Installing",
    running: "Installing…",
    failedStart: "Failed to start install",
  },
  upgrade: {
    title: "Upgrading",
    running: "Upgrading…",
    failedStart: "Failed to start upgrade",
  },
  remove: {
    title: "Removing",
    running: "Removing…",
    failedStart: "Failed to start remove",
  },
};

function appName(apps: AppStatus[], id: string): string {
  return apps.find((app) => app.id === id)?.name ?? id;
}

export function AppJobModal({
  action,
  appIds,
  apps,
  extraVars = {},
  queuePosition,
  queueTotal,
  onClose,
  onFinished,
}: Props) {
  const queryClient = useQueryClient();
  const logRef = useRef<HTMLPreElement>(null);
  const labels = ACTION_LABELS[action];
  const runId = useId();

  const activeAppId = appIds[0] ?? null;

  const extraVarsKey = useMemo(
    () => JSON.stringify(extraVars, Object.keys(extraVars).sort()),
    [extraVars],
  );

  const jobStart = useQuery({
    queryKey: ["app-job-start", action, activeAppId, runId, extraVarsKey],
    queryFn: () => startAppJob(activeAppId!, action, { extra_vars: extraVars }),
    enabled: activeAppId != null,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });

  const jobId = jobStart.data?.job_id;

  const job = useQuery({
    queryKey: ["job", jobId],
    enabled: Boolean(jobId),
    queryFn: () => getJob(jobId!),
    retry: (failureCount, error) =>
      error instanceof ApiError && error.status === 404 ? failureCount < 5 : false,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status && TERMINAL.includes(status)) {
        return false;
      }
      return 1000;
    },
  });

  const logs = useQuery({
    queryKey: ["job-logs", jobId],
    enabled: Boolean(jobId) && !job.isError,
    queryFn: () => getJobLogs(jobId!),
    retry: (failureCount, error) =>
      error instanceof ApiError && error.status === 404 ? failureCount < 5 : false,
    refetchInterval: () => {
      const status = job.data?.status;
      if (status && TERMINAL.includes(status)) {
        return false;
      }
      return 1000;
    },
  });

  const title = useMemo(() => {
    if (!activeAppId) {
      return `${labels.title} apps`;
    }
    return `${labels.title} ${appName(apps, activeAppId)}`;
  }, [activeAppId, labels.title, apps]);

  const status = job.data?.status ?? (jobStart.isPending ? "queued" : undefined);
  const isTerminal = status != null && TERMINAL.includes(status);
  const canClose = isTerminal || jobStart.isError || job.isError;

  const handleClose = useCallback(() => {
    if (status === "succeeded" && activeAppId) {
      void queryClient.invalidateQueries({ queryKey: ["apps"] });
      onFinished(activeAppId);
      return;
    }
    onClose();
  }, [activeAppId, onClose, onFinished, queryClient, status]);

  useEffect(() => {
    if (!logRef.current) {
      return;
    }
    logRef.current.scrollTop = logRef.current.scrollHeight;
  }, [logs.data?.lines.length]);

  const isError = jobStart.isError || job.isError;

  const statusLabel = (() => {
    if (jobStart.isError) {
      return labels.failedStart;
    }
    if (job.isError) {
      return job.error instanceof ApiError && job.error.status === 404
        ? "Install job unavailable"
        : "Job status unavailable";
    }
    switch (status) {
      case "queued":
        return "Queued";
      case "running":
        return "Running";
      case "succeeded":
        return "Succeeded";
      case "failed":
        return "Failed";
      default:
        return jobStart.isFetching ? "Starting…" : "Waiting to start…";
    }
  })();

  const errorMessage = (() => {
    if (jobStart.error) {
      return jobStart.error.message;
    }
    if (job.error instanceof ApiError && job.error.status === 404) {
      return "The install job is no longer on the server (it may have been started before a restart). Close this dialog and start the install again.";
    }
    if (job.error) {
      return job.error.message;
    }
    return job.data?.error;
  })();
  const logText = (logs.data?.lines ?? []).join("\n");

  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    if (!logText) {
      return;
    }
    try {
      await navigator.clipboard.writeText(logText);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  }, [logText]);

  useEffect(() => {
    if (!copied) {
      return;
    }
    const timer = window.setTimeout(() => setCopied(false), 2000);
    return () => window.clearTimeout(timer);
  }, [copied]);

  const statusVariant =
    status === "succeeded"
      ? "success"
      : isError || status === "failed"
        ? "destructive"
        : status === "running"
          ? "default"
          : "secondary";
  const StatusIcon =
    status === "succeeded"
      ? IconCheck
      : isError || status === "failed"
        ? IconAlertTriangle
        : IconLoader2;
  const statusIconClassName =
    status === "succeeded" || isError || status === "failed" ? undefined : "animate-spin";

  return (
    <Dialog open onOpenChange={(open) => !open && handleClose()}>
      <DialogContent
        showCloseButton={canClose}
        onEscapeKeyDown={(event) => {
          if (!canClose) {
            event.preventDefault();
          }
        }}
        onInteractOutside={(event) => event.preventDefault()}
        className="flex h-full max-h-[90vh] w-full sm:!max-w-[45vw] flex-col gap-0 p-0"
      >
        <DialogHeader className="gap-1 border-b border-border px-5 py-4">
          <div className="flex items-center gap-2">
            <DialogTitle className="min-w-0 truncate text-base">{title}</DialogTitle>
            <Badge variant={statusVariant}>
              <StatusIcon data-icon="inline-start" className={statusIconClassName} />
              {statusLabel}
            </Badge>
          </div>
          {queueTotal > 1 ? (
            <DialogDescription>
              App {queuePosition} of {queueTotal}
            </DialogDescription>
          ) : (
            <DialogDescription className="sr-only">Live output for {title}</DialogDescription>
          )}
          {errorMessage && (
            <p className="min-w-0 truncate text-sm text-destructive">{errorMessage}</p>
          )}
        </DialogHeader>

        <pre
          ref={logRef}
          className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto bg-muted px-5 py-3 font-mono text-xs leading-relaxed text-muted-foreground"
        >
          {logText || (canClose ? "No job output was captured." : "Waiting for job output…")}
        </pre>

        <DialogFooter className="border-t border-border px-5 py-4 sm:justify-between">
          <Button variant="outline" onClick={handleCopy} disabled={!logText}>
            {copied ? (
              <IconCheck data-icon="inline-start" />
            ) : (
              <IconCopy data-icon="inline-start" />
            )}
            {copied ? "Copied!" : "Copy log"}
          </Button>
          <Button variant="outline" onClick={handleClose} disabled={!canClose}>
            {canClose ? "Close" : labels.running}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
