import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { VnstatPanel } from "@/components/vnstat-panel";

describe("VnstatPanel", () => {
  it("shows missing message when vnstat is not installed", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          installed: false,
          message: "vnstat is not installed on this system.",
        }),
      }),
    );

    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={client}>
        <VnstatPanel />
      </QueryClientProvider>,
    );

    expect(await screen.findByText(/vnstat is not installed/i)).toBeInTheDocument();
  });

  it("renders traffic tables when installed", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          installed: true,
          version: "2.12",
          months: [{ label: "2026-05", rx_bytes: 1000, tx_bytes: 500 }],
          days: [{ label: "2026-05-31", rx_bytes: 100, tx_bytes: 50 }],
        }),
      }),
    );

    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={client}>
        <VnstatPanel />
      </QueryClientProvider>,
    );

    expect(await screen.findByText("Last 12 months")).toBeInTheDocument();
    expect(screen.getByText("2026-05")).toBeInTheDocument();
    expect(screen.getByText("2026-05-31")).toBeInTheDocument();
  });
});
