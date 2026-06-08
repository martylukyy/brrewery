import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useId, useMemo, useRef } from "react";

import {
  ApiError,
  getJob,
  getJobLogs,
  startPackageJob,
  type JobAction,
  type JobStatus,
  type PackageStatus,
} from "@/lib/api";

type Props = {
  action: JobAction;
  packageIds: string[];
  packages: PackageStatus[];
  extraVars?: Record<string, string>;
  queuePosition: number;
  queueTotal: number;
  onClose: () => void;
  onFinished: (packageId: string) => void;
};

const TERMINAL: JobStatus[] = ["succeeded", "failed"];

const ACTION_LABELS: Record<JobAction, { title: string; running: string; failedStart: string; output: string }> = {
  install: {
    title: "Installing",
    running: "Installing…",
    failedStart: "Failed to start install",
    output: "Ansible install output",
  },
  upgrade: {
    title: "Upgrading",
    running: "Upgrading…",
    failedStart: "Failed to start upgrade",
    output: "Ansible upgrade output",
  },
  remove: {
    title: "Removing",
    running: "Removing…",
    failedStart: "Failed to start remove",
    output: "Ansible remove output",
  },
};

function packageName(packages: PackageStatus[], id: string): string {
  return packages.find((pkg) => pkg.id === id)?.name ?? id;
}

export function PackageJobModal({
  action,
  packageIds,
  packages,
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

  const activePackageId = packageIds[0] ?? null;

  const extraVarsKey = useMemo(
    () => JSON.stringify(extraVars, Object.keys(extraVars).sort()),
    [extraVars],
  );

  const jobStart = useQuery({
    queryKey: ["package-job-start", action, activePackageId, runId, extraVarsKey],
    queryFn: () => startPackageJob(activePackageId!, action, { extra_vars: extraVars }),
    enabled: activePackageId != null,
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
    if (!activePackageId) {
      return `${labels.title} packages`;
    }
    return `${labels.title} ${packageName(packages, activePackageId)}`;
  }, [activePackageId, labels.title, packages]);

  const status = job.data?.status ?? (jobStart.isPending ? "queued" : undefined);
  const isTerminal = status != null && TERMINAL.includes(status);
  const canClose = isTerminal || jobStart.isError || job.isError;

  const handleClose = useCallback(() => {
    if (status === "succeeded" && activePackageId) {
      void queryClient.invalidateQueries({ queryKey: ["packages"] });
      onFinished(activePackageId);
      return;
    }
    onClose();
  }, [activePackageId, onClose, onFinished, queryClient, status]);

  useEffect(() => {
    if (!logRef.current) {
      return;
    }
    logRef.current.scrollTop = logRef.current.scrollHeight;
  }, [logs.data?.lines.length]);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape" && canClose) {
        handleClose();
      }
    }

    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [canClose, handleClose]);

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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        type="button"
        className="absolute inset-0 bg-black/60"
        aria-label="Close package job dialog"
        onClick={canClose ? handleClose : undefined}
        disabled={!canClose}
      />

      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="package-job-title"
        className="relative z-10 flex max-h-[90%] w-full max-w-[90%] flex-col rounded-lg border border-zinc-700 bg-zinc-900 shadow-xl"
      >
        <div className="border-b border-zinc-800 px-5 py-4">
          <h2 id="package-job-title" className="text-lg font-semibold text-zinc-100">
            {title}
          </h2>
          <p className="mt-1 text-sm text-zinc-400">
            {queueTotal > 1 ? `Package ${queuePosition} of ${queueTotal}` : labels.output}
          </p>
        </div>

        <div className="flex items-center gap-3 border-b border-zinc-800 px-5 py-3">
          <span
            className={
              status === "succeeded"
                ? "rounded-full bg-emerald-900/50 px-2 py-0.5 text-xs text-emerald-300"
                : status === "failed" || jobStart.isError || job.isError
                  ? "rounded-full bg-red-900/50 px-2 py-0.5 text-xs text-red-300"
                  : "rounded-full bg-amber-900/40 px-2 py-0.5 text-xs text-amber-200"
            }
          >
            {statusLabel}
          </span>
          {errorMessage && (
            <p className="min-w-0 truncate text-sm text-red-400">{errorMessage}</p>
          )}
        </div>

        <pre
          ref={logRef}
          className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto px-5 py-3 text-xs leading-relaxed text-zinc-300"
        >
          {logText || (canClose ? "No job output was captured." : "Waiting for job output…")}
        </pre>

        <div className="flex justify-end gap-2 border-t border-zinc-800 px-5 py-4">
          <button
            type="button"
            className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-50"
            onClick={handleClose}
            disabled={!canClose}
          >
            {canClose ? "Close" : labels.running}
          </button>
        </div>
      </div>
    </div>
  );
}
