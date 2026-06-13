import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SysctlModal } from "@/components/sysctl-modal";
import * as api from "@/lib/api";
import type { SysctlReport } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  getSysctl: vi.fn(),
  applySysctl: vi.fn(),
  verifyPassword: vi.fn(),
  ApiError: class ApiError extends Error {
    status: number;
    constructor(message: string, status: number) {
      super(message);
      this.status = status;
    }
  },
}));

const report: SysctlReport = {
  writable: true,
  settings: [
    {
      key: "vm.swappiness",
      label: "Swappiness",
      description: "How aggressively the kernel swaps.",
      group: "Memory",
      kind: "integer",
      recommended: "10",
      value: "60",
      available: true,
    },
    {
      key: "net.ipv4.tcp_congestion_control",
      label: "TCP congestion control",
      description: "Congestion control algorithm.",
      group: "Network",
      kind: "enum",
      recommended: "bbr",
      choices: ["bbr", "cubic"],
      value: "cubic",
      available: true,
    },
    {
      key: "kernel.sched_migration_cost_ns",
      label: "Scheduler migration cost",
      description: "Cache-hot time before migration.",
      group: "Kernel",
      kind: "integer",
      recommended: "5000000",
      value: "",
      available: false,
    },
  ],
};

function renderModal() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <SysctlModal onClose={() => {}} />
    </QueryClientProvider>,
  );
}

describe("SysctlModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getSysctl).mockResolvedValue(report);
    vi.mocked(api.applySysctl).mockResolvedValue(report);
    vi.mocked(api.verifyPassword).mockResolvedValue(undefined);
  });

  it("seeds inputs from the live values", async () => {
    renderModal();
    expect(await screen.findByText("Swappiness")).toBeInTheDocument();
    expect((screen.getByLabelText("Swappiness") as HTMLInputElement).value).toBe("60");
  });

  it("leaves the input empty for unavailable parameters", async () => {
    renderModal();
    await screen.findByText("Scheduler migration cost");
    const input = screen.getByLabelText("Scheduler migration cost") as HTMLInputElement;
    // Not seeded with the recommended value; shown empty and disabled instead.
    expect(input.value).toBe("");
    expect(input).toBeDisabled();
  });

  it("shows only Upload patch and Apply in the footer", async () => {
    renderModal();
    await screen.findByText("Swappiness");
    expect(screen.getByRole("button", { name: "Upload patch" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Apply" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Close" })).not.toBeInTheDocument();
  });

  it("prompts for the password when applying", async () => {
    const user = userEvent.setup();
    renderModal();
    await screen.findByText("Swappiness");

    expect(screen.queryByText("Confirm your password")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Apply" }));
    expect(await screen.findByText("Confirm your password")).toBeInTheDocument();
  });

  it("applies the edited values after confirming the password", async () => {
    const user = userEvent.setup();
    renderModal();
    await screen.findByText("Swappiness");

    const swappiness = screen.getByLabelText("Swappiness");
    await user.clear(swappiness);
    await user.type(swappiness, "10");

    await user.click(screen.getByRole("button", { name: "Apply" }));
    await user.type(await screen.findByLabelText("Account password"), "secret");
    await user.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => expect(api.applySysctl).toHaveBeenCalledTimes(1));
    expect(api.verifyPassword).toHaveBeenCalledWith("secret");
    expect(api.applySysctl).toHaveBeenCalledWith({
      password: "secret",
      values: {
        "vm.swappiness": "10",
        "net.ipv4.tcp_congestion_control": "cubic",
      },
    });
    expect(await screen.findByText("Settings applied.")).toBeInTheDocument();
  });

  it("applies an uploaded patch after confirming the password", async () => {
    const user = userEvent.setup();
    const { container } = renderModal();
    await screen.findByText("Swappiness");

    const file = new File(
      ["# brrewery tuning\nvm.swappiness = 5\nnet.ipv4.tcp_congestion_control = bbr\nkernel.unknown = 1\n"],
      "tune.conf",
      { type: "text/plain" },
    );
    const fileInput = container.querySelector("input[type=file]") as HTMLInputElement;
    await user.upload(fileInput, file);

    await user.type(await screen.findByLabelText("Account password"), "secret");
    await user.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => expect(api.applySysctl).toHaveBeenCalledTimes(1));
    // Only catalog keys from the patch are applied; the unknown key is ignored.
    expect(api.applySysctl).toHaveBeenCalledWith({
      password: "secret",
      values: {
        "vm.swappiness": "5",
        "net.ipv4.tcp_congestion_control": "bbr",
      },
    });
  });
});
