import { describe, expect, it } from "vitest";

import { normalizeSystemInfo, type SystemInfoRaw } from "@/lib/api";

const base: SystemInfoRaw = {
  hostname: "host",
  uptime_seconds: 1,
  cpu_count: 1,
  cpu_name: "cpu",
  cpu_percent: 0,
  load: { "1m": 0, "5m": 0, "15m": 0 },
  memory: {
    total_bytes: 1,
    available_bytes: 1,
    used_bytes: 0,
    used_percent: 0,
  },
  network: { rx_bytes: 0, tx_bytes: 0 },
  disk_io: { read_bytes: 0, write_bytes: 0, read_ops: 0, write_ops: 0 },
  disks: [],
};

describe("normalizeSystemInfo", () => {
  it("maps legacy disk to disks", () => {
    const legacy = {
      ...base,
      disk: {
        mount: "/",
        total_bytes: 100,
        used_bytes: 40,
        available_bytes: 60,
        used_percent: 40,
      },
    };

    const info = normalizeSystemInfo(legacy);
    expect(info.disks).toHaveLength(1);
    expect(info.disks[0]?.mount).toBe("/");
  });

  it("maps legacy disk_io_busy_percent to first disk", () => {
    const legacy = {
      ...base,
      disk_io_busy_percent: 12.5,
      disk: {
        mount: "/",
        total_bytes: 100,
        used_bytes: 40,
        available_bytes: 60,
        used_percent: 40,
      },
    };

    const info = normalizeSystemInfo(legacy);
    expect(info.disks[0]?.io_busy_percent).toBe(12.5);
  });
});
