import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Dashboard } from "@/components/dashboard";

describe("Dashboard", () => {
  it("renders system metrics", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          hostname: "brrewery-host",
          uptime_seconds: 3600,
          cpu_count: 4,
          cpu_name: "Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz",
          cpu_percent: 42.5,
          load: { "1m": 0.5, "5m": 0.4, "15m": 0.3 },
          memory: {
            total_bytes: 8_000_000_000,
            available_bytes: 4_000_000_000,
            used_bytes: 4_000_000_000,
            used_percent: 50,
          },
          disks: [
            {
              mount: "/",
              total_bytes: 100_000_000_000,
              used_bytes: 40_000_000_000,
              available_bytes: 60_000_000_000,
              used_percent: 40,
              io_busy_percent: 3.2,
              io_read_bytes: 10_000_000,
              io_write_bytes: 5_000_000,
            },
            {
              mount: "/mnt/storage",
              total_bytes: 50_000_000_000,
              used_bytes: 10_000_000_000,
              available_bytes: 40_000_000_000,
              used_percent: 20,
              io_busy_percent: 1.5,
              io_read_bytes: 2_000_000,
              io_write_bytes: 1_000_000,
            },
          ],
          network: { rx_bytes: 1_000_000, tx_bytes: 500_000 },
          disk_io: {
            read_bytes: 10_000_000,
            write_bytes: 5_000_000,
            read_ops: 1000,
            write_ops: 500,
          },
        }),
      }),
    );

    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={client}>
        <Dashboard />
      </QueryClientProvider>,
    );

    expect(await screen.findByText(/brrewery-host/)).toBeInTheDocument();
    expect(screen.getByText("brrewery")).toBeInTheDocument();
    expect(screen.getByText("CPU")).toBeInTheDocument();
    expect(screen.getByText("42.5%")).toBeInTheDocument();
    expect(screen.getByText("Load average")).toBeInTheDocument();
    expect(screen.getByText("Memory")).toBeInTheDocument();
    expect(screen.getByText("1m")).toBeInTheDocument();
    expect(screen.getByText("Network throughput")).toBeInTheDocument();
    expect(screen.getByText("/")).toBeInTheDocument();
    expect(screen.getByText("/mnt/storage")).toBeInTheDocument();
    expect(screen.getAllByText("Disk usage")).toHaveLength(2);
    expect(screen.getAllByText("I/O busy")).toHaveLength(2);
    expect(screen.getByText("3.20%")).toBeInTheDocument();
    expect(screen.getByText("/ throughput")).toBeInTheDocument();
    expect(screen.getByText("/mnt/storage throughput")).toBeInTheDocument();
  });
});
