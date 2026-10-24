import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
          days: [
            { label: "2026-05-31", rx_bytes: 100, tx_bytes: 50 },
            { label: "2026-05-30", rx_bytes: 300, tx_bytes: 100 },
          ],
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

    expect(await screen.findByText("2026-05-31")).toBeInTheDocument();

    const user = userEvent.setup();

    await user.click(screen.getByRole("combobox", { name: "vnStat range" }));
    await user.click(await screen.findByRole("option", { name: "Last 12 months" }));
    expect(screen.getByText("2026-05")).toBeInTheDocument();

    await user.click(screen.getByRole("combobox", { name: "vnStat range" }));
    await user.click(await screen.findByRole("option", { name: "Top 10 days overall" }));
    const top10Rows = screen.getAllByRole("row").slice(1);
    expect(top10Rows[0]).toHaveTextContent("2026-05-30");
    expect(top10Rows[1]).toHaveTextContent("2026-05-31");
  });

  it("renders the period summary above the range table", async () => {
    const GiB = 1024 ** 3;
    const PiB = 1024 ** 5;
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
          summary: {
            this_hour: { rx_bytes: GiB, tx_bytes: 3 * GiB },
            last_hour: { rx_bytes: 2 * GiB, tx_bytes: 2 * GiB },
            this_day: { rx_bytes: 10 * GiB, tx_bytes: 20 * GiB },
            this_month: { rx_bytes: 100 * GiB, tx_bytes: 200 * GiB },
            all_time: { rx_bytes: PiB / 2, tx_bytes: 4 * PiB },
          },
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

    expect(await screen.findByText("This Hour")).toBeInTheDocument();

    const [summaryTable, rangeTable] = screen.getAllByRole("table");
    const summaryRows = within(summaryTable).getAllByRole("row").slice(1);
    expect(summaryRows.map((row) => within(row).getAllByRole("cell")[0].textContent)).toEqual([
      "This Hour",
      "Last Hour",
      "This Day",
      "This Month",
      "All Time",
    ]);
    expect(summaryRows[0]).toHaveTextContent("1.0 GiB3.0 GiB4.0 GiB");
    expect(summaryRows[4]).toHaveTextContent("512.0 TiB4.0 PiB4.5 PiB");

    expect(within(rangeTable).getByText("2026-05-31")).toBeInTheDocument();
  });
});
