import { useQuery, useQueryClient } from "@tanstack/react-query";
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
import { UPDATE_STATUS_QUERY_KEY } from "@/hooks/use-update-status";
import {
  ApiError,
  checkSession,
  finishSelfUpdate,
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
//                When an update is already installed (restart_pending), it
//                offers the restart directly instead.
//   running    — the job installs the release; logs are polled like app jobs.
//                The job ends in "succeeded" with the old process still
//                serving, so the operator sees the result and decides when to
//                restart via the "Restart brrewery" button.
//   restarting — the operator confirmed; the service restart is underway. The
//                in-memory sessions die with it, so the authenticated session
//                probe answering 401 (checkSession -> null) is the signal that
//                the new process is up.
//   done       — the new version is serving; the user must sign in again.
//   timeout    — the service never came back; advise checking it manually.
type Phase = "confirm" | "running" | "restarting" | "done" | "timeout";

// How long to wait for the restarted service before giving up.
const RESTART_DEADLINE_MS = 120_000;
const RESTART_POLL_MS = 2_000;

const TERMINAL = ["succeeded", "failed"];

export function UpdateModal({ status, onClose }: Props) {
  const queryClient = useQueryClient();
  const [phase, setPhase] = useState<Phase>("confirm");
  const [jobId, setJobId] = useState<string | null>(null);
  const [password, setPassword] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [restartRequesting, setRestartRequesting] = useState(false);
  const [restartError, setRestartError] = useState<string | null>(null);
  const logRef = useRef<HTMLPreElement>(null);

  const job = useQuery({
    queryKey: ["job", jobId],
    enabled: phase === "running" && jobId != null,
    queryFn: () => getJob(jobId!),
    retry: false,
    refetchInterval: (query) => {
      const jobStatus = query.state.data?.status;
      if ((jobStatus && TERMINAL.includes(jobStatus)) || query.state.error) {
        return false;
      }
      return 1000;
    },
  });

  const jobFailed = job.data?.status === "failed";
  const jobSucceeded = job.data?.status === "succeeded";

  // A succeeded install flips restart_pending on the server. Refetch the
  // cached update status so closing and reopening the modal lands on the
  // "finish the installed update" screen instead of a second install.
  useEffect(() => {
    if (jobSucceeded) {
      void queryClient.invalidateQueries({ queryKey: UPDATE_STATUS_QUERY_KEY });
    }
  }, [jobSucceeded, queryClient]);

  const logs = useQuery({
    queryKey: ["job-logs", jobId],
    enabled: phase === "running" && jobId != null,
    queryFn: () => getJobLogs(jobId!),
    retry: false,
    refetchInterval: (query) => {
      const jobStatus = job.data?.status;
      if ((jobStatus && TERMINAL.includes(jobStatus)) || query.state.error) {
        return false;
      }
      return 1000;
    },
  });

  // Watch for the restarted process: while the old one is still up the
  // session probe answers with version info; once the new process serves, the
  // in-memory session is gone and the probe returns null (401). Errors in
  // between are the downtime window.
  useEffect(() => {
    if (phase !== "restarting") {
      return;
    }
    const deadline = Date.now() + RESTART_DEADLINE_MS;
    let cancelled = false;
    const settle = (restarted: boolean) => {
      if (cancelled) {
        return;
      }
      if (restarted) {
        setPhase("done");
      } else if (Date.now() > deadline) {
        setPhase("timeout");
      }
    };
    const timer = window.setInterval(() => {
      checkSession().then(
        (session) => settle(session === null),
        () => settle(false),
      );
    }, RESTART_POLL_MS);
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
      setPhase("running");
    } catch (error) {
      setFormError(error instanceof Error ? error.message : "Failed to start the update.");
    } finally {
      setSubmitting(false);
    }
  }

  // The operator's explicit go-ahead: only this restarts the service.
  async function handleRestart() {
    setRestartRequesting(true);
    setRestartError(null);
    try {
      await finishSelfUpdate();
      setPhase("restarting");
    } catch (error) {
      // The restart tears the server down, which can kill this very request:
      // a network-level failure or a gateway error from nginx here means the
      // restart is underway, not that it failed. Only a real answer from the
      // backend (409/500) is an error worth showing.
      const restartUnderway =
        !(error instanceof ApiError) ||
        error.status === 502 ||
        error.status === 503 ||
        error.status === 504;
      if (restartUnderway) {
        setPhase("restarting");
        return;
      }
      setRestartError(error.message || "Failed to restart brrewery.");
    } finally {
      setRestartRequesting(false);
    }
  }

  // The dialog must not be dismissable while work is in flight. After a
  // successful install it can be closed — the update is staged on disk and
  // the restart can happen later.
  const canClose =
    phase === "confirm" || phase === "timeout" || jobFailed || jobSucceeded || job.isError;

  function handleClose() {
    if (canClose) {
      onClose();
    }
  }

  const versionLabel = status.latest_version ?? "the latest version";
  const logText = (logs.data?.lines ?? []).join("\n");

  const restartButton = (
    <Button onClick={handleRestart} disabled={restartRequesting}>
      {restartRequesting ? <Spinner data-icon="inline-start" /> : <IconRefresh data-icon="inline-start" />}
      Restart brrewery
    </Button>
  );

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
        {phase === "confirm" && status.restart_pending ? (
          <>
            <DialogHeader className="gap-1 border-b border-border px-5 py-4">
              <DialogTitle className="text-base">Finish the installed update</DialogTitle>
              <DialogDescription>
                An update has already been installed and is waiting for a restart. Restart
                brrewery to start using it — you will be signed out.
              </DialogDescription>
              {restartError && <p className="text-sm text-destructive">{restartError}</p>}
            </DialogHeader>
            <DialogFooter className="border-t border-border px-5 py-4">
              <Button type="button" variant="outline" onClick={onClose} disabled={restartRequesting}>
                Cancel
              </Button>
              {restartButton}
            </DialogFooter>
          </>
        ) : phase === "confirm" ? (
          <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
            <DialogHeader className="gap-1 border-b border-border px-5 py-4">
              <DialogTitle className="text-base">Update brrewery</DialogTitle>
              <DialogDescription>
                Update from version {status.current_version} to {versionLabel}. The update is
                installed in the background; brrewery keeps running until you confirm the
                restart afterwards.
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
                <UpdatePhaseBadge
                  phase={phase}
                  failed={jobFailed || job.isError}
                  installed={jobSucceeded}
                />
              </div>
              <DialogDescription className="sr-only">
                Live output for the brrewery update
              </DialogDescription>
              {jobSucceeded && phase === "running" && (
                <p className="text-sm text-green-700 dark:text-green-300">
                  Update installed successfully. Restart brrewery to start using{" "}
                  {versionLabel} — you will be signed out.
                </p>
              )}
              {jobFailed && job.data?.error && (
                <p className="min-w-0 truncate text-sm text-destructive">{job.data.error}</p>
              )}
              {job.isError && (
                <p className="text-sm text-destructive">
                  Lost track of the update job: {job.error.message}
                </p>
              )}
              {restartError && <p className="text-sm text-destructive">{restartError}</p>}
              {phase === "timeout" && (
                <p className="text-sm text-destructive">
                  brrewery did not come back within {RESTART_DEADLINE_MS / 1000} seconds. The
                  restart may still be finishing — check the server and reload this page.
                </p>
              )}
            </DialogHeader>

            <pre
              ref={logRef}
              className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto bg-muted px-5 py-3 font-mono text-xs leading-relaxed text-muted-foreground"
            >
              {phase === "done"
                ? `${logText ? logText + "\n" : ""}Update installed — sign in to continue.`
                : phase === "restarting"
                  ? `${logText ? logText + "\n" : ""}Restarting brrewery…`
                  : logText || "Waiting for job output…"}
            </pre>

            <DialogFooter className="border-t border-border px-5 py-4">
              {phase === "done" ? (
                <Button onClick={() => window.location.reload()}>Sign in again</Button>
              ) : phase === "restarting" ? (
                <Button variant="outline" disabled>
                  Restarting…
                </Button>
              ) : jobSucceeded ? (
                <>
                  <Button variant="outline" onClick={onClose} disabled={restartRequesting}>
                    Close
                  </Button>
                  {restartButton}
                </>
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

function UpdatePhaseBadge({
  phase,
  failed,
  installed,
}: {
  phase: Phase;
  failed: boolean;
  installed: boolean;
}) {
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
      return installed ? (
        <Badge variant="success">
          <IconCheck data-icon="inline-start" />
          Installed
        </Badge>
      ) : (
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
