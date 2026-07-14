import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UpdateModal } from "@/components/update-modal";
import * as api from "@/lib/api";
import type { AppJob, UpdateStatus } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  verifyPassword: vi.fn(),
  startSelfUpdate: vi.fn(),
  getJob: vi.fn(),
  getJobLogs: vi.fn(),
  checkHealth: vi.fn(),
  ApiError: class ApiError extends Error {
    status: number;
    path: string;
    constructor(message: string, status: number, path = "") {
      super(message);
      this.status = status;
      this.path = path;
    }
  },
}));

const status: UpdateStatus = {
  current_version: "1.0.0",
  latest_version: "1.1.0",
  latest_tag: "v1.1.0",
  update_available: true,
};

const runningJob: AppJob = {
  id: "job-1",
  app_id: "brrewery",
  action: "self-update",
  status: "running",
  started_at: "2026-01-01T00:00:00Z",
};

function renderModal(onClose = () => {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <UpdateModal status={status} onClose={onClose} />
    </QueryClientProvider>,
  );
}

async function submitPassword(password = "password123") {
  const user = userEvent.setup();
  await user.type(screen.getByLabelText("Password"), password);
  await user.click(screen.getByRole("button", { name: "Install update" }));
}

describe("UpdateModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.verifyPassword).mockResolvedValue(undefined);
    vi.mocked(api.startSelfUpdate).mockResolvedValue({ job_id: "job-1" });
    vi.mocked(api.getJob).mockResolvedValue(runningJob);
    vi.mocked(api.getJobLogs).mockResolvedValue({ lines: [] });
    vi.mocked(api.checkHealth).mockResolvedValue(false);
  });

  it("names both versions in the confirmation", () => {
    renderModal();

    expect(
      screen.getByText(/Update from version 1\.0\.0 to 1\.1\.0/),
    ).toBeInTheDocument();
  });

  it("requires a password before starting", async () => {
    renderModal();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Install update" }));

    expect(await screen.findByText("Account password is required.")).toBeInTheDocument();
    expect(api.verifyPassword).not.toHaveBeenCalled();
    expect(api.startSelfUpdate).not.toHaveBeenCalled();
  });

  it("rejects a wrong password without starting the update", async () => {
    vi.mocked(api.verifyPassword).mockRejectedValue(
      new api.ApiError("Unauthorized", 401, "/auth/verify-password"),
    );
    renderModal();

    await submitPassword("nope");

    expect(await screen.findByText("Incorrect password.")).toBeInTheDocument();
    expect(api.startSelfUpdate).not.toHaveBeenCalled();
  });

  it("starts the job and streams its logs", async () => {
    vi.mocked(api.getJobLogs).mockResolvedValue({
      lines: ["downloading brrewery 1.1.0", "verifying checksum"],
    });
    renderModal();

    await submitPassword();

    expect(await screen.findByText("Installing")).toBeInTheDocument();
    expect(await screen.findByText(/verifying checksum/)).toBeInTheDocument();
    expect(api.startSelfUpdate).toHaveBeenCalledWith("password123");
    // The dialog is locked while the update runs.
    expect(screen.getByRole("button", { name: "Updating…" })).toBeDisabled();
  });

  it("surfaces a failed job and allows closing", async () => {
    vi.mocked(api.getJob).mockResolvedValue({
      ...runningJob,
      status: "failed",
      error: "checksum mismatch",
    });
    const onClose = vi.fn();
    renderModal(onClose);

    await submitPassword();

    expect(await screen.findByText("Failed")).toBeInTheDocument();
    expect(await screen.findByText("checksum mismatch")).toBeInTheDocument();

    // Both the dialog's X and the footer button read "Close" once the job
    // fails; the footer one is rendered last.
    const closeButtons = screen.getAllByRole("button", { name: "Close" });
    const user = userEvent.setup();
    await user.click(closeButtons[closeButtons.length - 1]);
    expect(onClose).toHaveBeenCalled();
  });

  it("treats job-poll failures as the restart and finishes on a healthy probe", async () => {
    // The service went down for the restart: job polling fails, then /health
    // answers once the new process is up.
    vi.mocked(api.getJob).mockRejectedValue(new api.ApiError("Bad Gateway", 502, "/jobs/job-1"));
    vi.mocked(api.checkHealth).mockResolvedValue(true);
    renderModal();

    await submitPassword();

    expect(await screen.findByText("Restarting")).toBeInTheDocument();
    expect(
      await screen.findByRole("button", { name: "Sign in again" }, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.getByText("Updated")).toBeInTheDocument();
  }, 10_000);
});
