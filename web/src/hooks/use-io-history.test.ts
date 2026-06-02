import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useIOHistory } from "@/hooks/use-io-history";
import type { SystemInfo } from "@/lib/api";

const base: SystemInfo = {
  hostname: "host",
  uptime_seconds: 100,
  cpu_count: 4,
  cpu_name: "Test CPU",
  cpu_percent: 0,
  disk_io_busy_percent: 0,
  load: { "1m": 0, "5m": 0, "15m": 0 },
  memory: {
    total_bytes: 1,
    available_bytes: 1,
    used_bytes: 0,
    used_percent: 0,
  },
  disk: {
    mount: "/",
    total_bytes: 1,
    used_bytes: 0,
    available_bytes: 1,
    used_percent: 0,
  },
  network: { rx_bytes: 1000, tx_bytes: 500 },
  disk_io: { read_bytes: 2000, write_bytes: 1000, read_ops: 10, write_ops: 5 },
};

describe("useIOHistory", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("computes rates between samples", async () => {
    vi.useFakeTimers();
    const start = new Date("2026-01-01T00:00:00Z");
    vi.setSystemTime(start);

    const { result, rerender } = renderHook(
      ({ info }: { info: SystemInfo | undefined }) => useIOHistory(info),
      { initialProps: { info: base } },
    );

    expect(result.current).toHaveLength(0);

    act(() => {
      vi.setSystemTime(new Date(start.getTime() + 5000));
      rerender({
        info: {
          ...base,
          network: { rx_bytes: 6000, tx_bytes: 2500 },
          disk_io: { read_bytes: 12_000, write_bytes: 6000, read_ops: 20, write_ops: 8 },
        },
      });
    });

    expect(result.current).toHaveLength(1);
    expect(result.current[0]?.rxPerSec).toBe(1000);
    expect(result.current[0]?.txPerSec).toBe(400);
    expect(result.current[0]?.readPerSec).toBe(2000);
    expect(result.current[0]?.writePerSec).toBe(1000);
  });
});
