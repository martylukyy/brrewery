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
  finishSelfUpdate: vi.fn(),
  getJob: vi.fn(),
  getJobLogs: vi.fn(),
  checkSession: vi.fn(),
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

function renderModal(onClose = () => {}, modalStatus: UpdateStatus = status) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <UpdateModal status={modalStatus} onClose={onClose} />
    </QueryClientProvider>,
  );
  return client;
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
    vi.mocked(api.finishSelfUpdate).mockResolvedValue({ status: "restarting" });
    vi.mocked(api.getJob).mockResolvedValue(runningJob);
    vi.mocked(api.getJobLogs).mockResolvedValue({ lines: [] });
    // Old process still serving: the session probe answers with version info.
    vi.mocked(api.checkSession).mockResolvedValue({
      version: "1.0.0",
      commit: "",
      date: "",
    });
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

  it("shows success and only restarts after the operator confirms", async () => {
    vi.mocked(api.getJob).mockResolvedValue({ ...runningJob, status: "succeeded" });
    renderModal();

    await submitPassword();

    // The install finished but nothing restarts on its own.
    expect(await screen.findByText("Installed")).toBeInTheDocument();
    expect(
      await screen.findByText(/Update installed successfully/),
    ).toBeInTheDocument();
    expect(api.finishSelfUpdate).not.toHaveBeenCalled();

    // New process comes up right after the restart: session probe answers 401.
    vi.mocked(api.checkSession).mockResolvedValue(null);

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Restart brrewery" }));
    expect(api.finishSelfUpdate).toHaveBeenCalledOnce();

    expect(await screen.findByText("Restarting")).toBeInTheDocument();
    expect(
      await screen.findByRole("button", { name: "Sign in again" }, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.getByText("Updated")).toBeInTheDocument();
  }, 10_000);

  it("treats a dropped restart request as the restart happening", async () => {
    vi.mocked(api.getJob).mockResolvedValue({ ...runningJob, status: "succeeded" });
    // The restart killed the connection before the response arrived.
    vi.mocked(api.finishSelfUpdate).mockRejectedValue(new TypeError("Failed to fetch"));
    renderModal();

    await submitPassword();
    await screen.findByText("Installed");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Restart brrewery" }));

    expect(await screen.findByText("Restarting")).toBeInTheDocument();
    expect(screen.queryByText(/Failed to restart/)).not.toBeInTheDocument();
  });

  it("shows a real backend refusal of the restart", async () => {
    vi.mocked(api.getJob).mockResolvedValue({ ...runningJob, status: "succeeded" });
    vi.mocked(api.finishSelfUpdate).mockRejectedValue(
      new api.ApiError("An update is still in progress", 409, "/update/restart"),
    );
    renderModal();

    await submitPassword();
    await screen.findByText("Installed");

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Restart brrewery" }));

    expect(await screen.findByText("An update is still in progress")).toBeInTheDocument();
    expect(screen.queryByText("Restarting")).not.toBeInTheDocument();
  });

  it("can be closed after a successful install without restarting", async () => {
    vi.mocked(api.getJob).mockResolvedValue({ ...runningJob, status: "succeeded" });
    const onClose = vi.fn();
    const client = renderModal(onClose);
    client.setQueryData(["update-status"], status);

    await submitPassword();
    await screen.findByText("Installed");

    // The install flipped restart_pending server-side; the cached status must
    // be refetched so reopening the modal offers the restart, not a reinstall.
    expect(client.getQueryState(["update-status"])?.isInvalidated).toBe(true);

    const closeButtons = screen.getAllByRole("button", { name: "Close" });
    const user = userEvent.setup();
    await user.click(closeButtons[closeButtons.length - 1]);

    expect(onClose).toHaveBeenCalled();
    expect(api.finishSelfUpdate).not.toHaveBeenCalled();
  });

  it("offers the restart directly when an update is already installed", async () => {
    renderModal(() => {}, { ...status, restart_pending: true });

    expect(screen.getByText(/already been installed/)).toBeInTheDocument();
    expect(screen.queryByLabelText("Password")).not.toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Restart brrewery" }));
    expect(api.finishSelfUpdate).toHaveBeenCalledOnce();
    expect(await screen.findByText("Restarting")).toBeInTheDocument();
  });
});
