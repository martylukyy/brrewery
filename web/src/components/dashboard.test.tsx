import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Dashboard } from "@/components/dashboard";

// The charts load their history from the daemon, so the dashboard issues two
// kinds of request; answer each with the shape it expects.
const ioHistory = {
  start_ms: 0,
  end_ms: 300_000,
  bucket_seconds: 1,
  sample_seconds: 1,
  smoothing_seconds: 5,
  series: [
    { key: "read", points: [1000, 2000] },
    { key: "write", points: [500, 600] },
  ],
};

const systemInfo = {
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
      read_bytes: 10_000_000,
      write_bytes: 5_000_000,
      read_ops: 1000,
      write_ops: 500,
    },
    {
      mount: "/mnt/storage",
      total_bytes: 50_000_000_000,
      used_bytes: 10_000_000_000,
      available_bytes: 40_000_000_000,
      used_percent: 20,
      io_busy_percent: 1.5,
      read_bytes: 2_000_000,
      write_bytes: 1_000_000,
      read_ops: 200,
      write_ops: 100,
    },
  ],
  network: { rx_bytes: 1_000_000, tx_bytes: 500_000 },
};

describe("Dashboard", () => {
  it("renders system metrics", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((url: string) => ({
        ok: true,
        status: 200,
        json: async () =>
          String(url).includes("/system/io-history") ? ioHistory : systemInfo,
      })),
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
    expect(screen.getByText("CPU")).toBeInTheDocument();
    expect(screen.getByText("42.5%")).toBeInTheDocument();
    expect(screen.getByText("Load average")).toBeInTheDocument();
    expect(screen.getByText("Memory")).toBeInTheDocument();
    expect(screen.getByText("Uptime")).toBeInTheDocument();
    expect(screen.getByText("1h 0m")).toBeInTheDocument();
    expect(screen.getByText("1m")).toBeInTheDocument();
    expect(screen.getByText("Network throughput")).toBeInTheDocument();
    expect(screen.getByText("/")).toBeInTheDocument();
    expect(screen.getByText("/mnt/storage")).toBeInTheDocument();
    expect(screen.getAllByText("Disk usage")).toHaveLength(2);
    expect(screen.getAllByText("I/O busy")).toHaveLength(2);
    expect(screen.getByText("3.20%")).toBeInTheDocument();
    expect(screen.getByText("Disk throughput ( / )")).toBeInTheDocument();
    expect(screen.getByText("Disk throughput ( /mnt/storage )")).toBeInTheDocument();
  });
});
