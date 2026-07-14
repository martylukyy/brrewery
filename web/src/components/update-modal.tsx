import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { IconAlertTriangle, IconCheck, IconRefresh } from "@tabler/icons-react";

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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import {
  ApiError,
  checkHealth,
  getJob,
  getJobLogs,
  startSelfUpdate,
  verifyPassword,
  type UpdateStatus,
} from "@/lib/api";

type Props = {
  status: UpdateStatus;
  onClose: () => void;
};

// The modal walks through the update's whole lifecycle:
//   confirm    — password gate, mirrors the other privileged-action modals.
//   running    — the job installs the release; logs are polled like app jobs.
//   restarting — job polling started failing: the service is restarting (the
//                in-memory sessions die with it, so a 401 here means "server is
//                back, session is gone", not "wrong password"). Probe the
//                unauthenticated /health endpoint until it answers.
//   done       — the new version is up; the user must sign in again.
//   timeout    — /health never answered; advise checking the server manually.
type Phase = "confirm" | "running" | "restarting" | "done" | "timeout";

// How long to wait for the restarted service before giving up.
const RESTART_DEADLINE_MS = 120_000;
const HEALTH_POLL_MS = 2_000;

export function UpdateModal({ status, onClose }: Props) {
  // "restarting" is never stored: it is derived below from the job poll
  // failing, so state only ever moves confirm -> running -> done/timeout.
  const [storedPhase, setStoredPhase] = useState<Exclude<Phase, "restarting">>("confirm");
  const [jobId, setJobId] = useState<string | null>(null);
  const [password, setPassword] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const logRef = useRef<HTMLPreElement>(null);

  const job = useQuery({
    queryKey: ["job", jobId],
    enabled: storedPhase === "running" && jobId != null,
    queryFn: () => getJob(jobId!),
    retry: false,
    // Stop polling on the first failure — from there the /health probe below
    // takes over.
    refetchInterval: (query) => (query.state.error ? false : 1000),
  });

  const jobFailed = job.data?.status === "failed";

  const logs = useQuery({
    queryKey: ["job-logs", jobId],
    enabled: storedPhase === "running" && jobId != null,
    queryFn: () => getJobLogs(jobId!),
    retry: false,
    refetchInterval: (query) => (jobFailed || query.state.error ? false : 1000),
  });

  // Job polling failing while the update runs means the process went down for
  // the restart (connection refused, a 5xx from nginx, or a 401 once the new
  // process is up and the old session is gone). Switch to probing /health.
  const phase: Phase = storedPhase === "running" && job.isError ? "restarting" : storedPhase;

  useEffect(() => {
    if (phase !== "restarting") {
      return;
    }
    const deadline = Date.now() + RESTART_DEADLINE_MS;
    let cancelled = false;
    const timer = window.setInterval(() => {
      void checkHealth().then((healthy) => {
        if (cancelled) {
          return;
        }
        if (healthy) {
          setStoredPhase("done");
        } else if (Date.now() > deadline) {
          setStoredPhase("timeout");
        }
      });
    }, HEALTH_POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [phase]);

  useEffect(() => {
    if (!logRef.current) {
      return;
    }
    logRef.current.scrollTop = logRef.current.scrollHeight;
  }, [logs.data?.lines.length]);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    if (!password.trim()) {
      setFormError("Account password is required.");
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      // Verify against the credential endpoint first: a wrong password there
      // is a 401 that does not sign the user out (see CREDENTIAL_PATHS).
      await verifyPassword(password);
    } catch (error) {
      setFormError(
        error instanceof ApiError && error.status === 401
          ? "Incorrect password."
          : error instanceof Error
            ? error.message
            : "Could not verify the password. Please try again.",
      );
      setSubmitting(false);
      return;
    }
    try {
      const response = await startSelfUpdate(password);
      setJobId(response.job_id);
      setStoredPhase("running");
    } catch (error) {
      setFormError(error instanceof Error ? error.message : "Failed to start the update.");
    } finally {
      setSubmitting(false);
    }
  }

  // The dialog must not be dismissable while the update is in flight — closing
  // it would not stop the install/restart, just hide it.
  const canClose = phase === "confirm" || phase === "timeout" || jobFailed;

  function handleClose() {
    if (canClose) {
      onClose();
    }
  }

  const versionLabel = status.latest_version ?? "the latest version";
  const logText = (logs.data?.lines ?? []).join("\n");

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
        className={
          phase === "confirm"
            ? "flex max-h-[90vh] flex-col gap-0 p-0 sm:max-w-md"
            : "flex h-full max-h-[90vh] w-full sm:!max-w-[45vw] flex-col gap-0 p-0"
        }
      >
        {phase === "confirm" ? (
          <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
            <DialogHeader className="gap-1 border-b border-border px-5 py-4">
              <DialogTitle className="text-base">Update brrewery</DialogTitle>
              <DialogDescription>
                Update from version {status.current_version} to {versionLabel}. brrewery
                restarts to finish the update and you will be signed out.
              </DialogDescription>
            </DialogHeader>

            <div className="scrollbar-zinc min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
              <div className="space-y-1">
                <Label htmlFor="update-password">Password</Label>
                <Input
                  id="update-password"
                  type="password"
                  value={password}
                  name="update-password"
                  autoComplete="current-password"
                  autoFocus
                  onChange={(event) => {
                    setPassword(event.target.value);
                    setFormError(null);
                  }}
                />
              </div>
              {formError && <p className="text-sm text-destructive">{formError}</p>}
            </div>

            <DialogFooter className="border-t border-border px-5 py-4">
              <Button type="button" variant="outline" onClick={onClose} disabled={submitting}>
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting && <Spinner data-icon="inline-start" />}
                Install update
              </Button>
            </DialogFooter>
          </form>
        ) : (
          <>
            <DialogHeader className="gap-1 border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <DialogTitle className="min-w-0 truncate text-base">
                  Updating brrewery to {versionLabel}
                </DialogTitle>
                <UpdatePhaseBadge phase={phase} failed={jobFailed} />
              </div>
              <DialogDescription className="sr-only">
                Live output for the brrewery update
              </DialogDescription>
              {jobFailed && job.data?.error && (
                <p className="min-w-0 truncate text-sm text-destructive">{job.data.error}</p>
              )}
              {phase === "timeout" && (
                <p className="text-sm text-destructive">
                  brrewery did not come back within {RESTART_DEADLINE_MS / 1000} seconds. The
                  update may still be finishing — check the server and reload this page.
                </p>
              )}
            </DialogHeader>

            <pre
              ref={logRef}
              className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto bg-muted px-5 py-3 font-mono text-xs leading-relaxed text-muted-foreground"
            >
              {phase === "done"
                ? `${logText ? logText + "\n" : ""}Update installed — sign in to continue.`
                : logText || "Waiting for job output…"}
            </pre>

            <DialogFooter className="border-t border-border px-5 py-4">
              {phase === "done" ? (
                <Button onClick={() => window.location.reload()}>Sign in again</Button>
              ) : (
                <Button variant="outline" onClick={handleClose} disabled={!canClose}>
                  {canClose ? "Close" : "Updating…"}
                </Button>
              )}
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

function UpdatePhaseBadge({ phase, failed }: { phase: Phase; failed: boolean }) {
  if (failed) {
    return (
      <Badge variant="destructive">
        <IconAlertTriangle data-icon="inline-start" />
        Failed
      </Badge>
    );
  }
  switch (phase) {
    case "running":
      return (
        <Badge variant="default">
          <Spinner data-icon="inline-start" />
          Installing
        </Badge>
      );
    case "restarting":
      return (
        <Badge variant="secondary">
          <IconRefresh data-icon="inline-start" className="animate-spin" />
          Restarting
        </Badge>
      );
    case "done":
      return (
        <Badge variant="success">
          <IconCheck data-icon="inline-start" />
          Updated
        </Badge>
      );
    case "timeout":
      return (
        <Badge variant="destructive">
          <IconAlertTriangle data-icon="inline-start" />
          Not responding
        </Badge>
      );
    default:
      return null;
  }
}
