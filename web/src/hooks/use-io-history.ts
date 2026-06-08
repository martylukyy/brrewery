import { useEffect, useRef, useState } from "react";

import type { NetworkCounters, SystemInfo } from "@/lib/api";
import { CHART_HISTORY_MAX_POINTS } from "@/lib/chart-interval";

export type NetworkSample = {
  rxPerSec: number;
  txPerSec: number;
};

export type DiskIOSample = {
  readPerSec: number;
  writePerSec: number;
};

type Snapshot = {
  at: number;
  network: NetworkCounters;
  diskByMount: Record<string, { readBytes: number; writeBytes: number }>;
};

function ratePerSec(current: number, previous: number, seconds: number): number {
  if (seconds <= 0 || current < previous) {
    return 0;
  }
  return (current - previous) / seconds;
}

export function useIOHistory(info: SystemInfo | undefined): {
  networkHistory: NetworkSample[];
  diskHistoryByMount: Record<string, DiskIOSample[]>;
} {
  const previous = useRef<Snapshot | null>(null);
  const [networkHistory, setNetworkHistory] = useState<NetworkSample[]>([]);
  const [diskHistoryByMount, setDiskHistoryByMount] = useState<Record<string, DiskIOSample[]>>({});

  useEffect(() => {
    if (!info) {
      return;
    }

    const now = Date.now();
    const prev = previous.current;
    const diskByMount = Object.fromEntries(
      (info.disks ?? []).map((disk) => [disk.mount, {
        readBytes: disk.read_bytes,
        writeBytes: disk.write_bytes,
      }]),
    );

    if (prev) {
      const seconds = (now - prev.at) / 1000;
      const networkSample: NetworkSample = {
        rxPerSec: ratePerSec(info.network.rx_bytes, prev.network.rx_bytes, seconds),
        txPerSec: ratePerSec(info.network.tx_bytes, prev.network.tx_bytes, seconds),
      };
      setNetworkHistory((current) => [...current, networkSample].slice(-CHART_HISTORY_MAX_POINTS));

      setDiskHistoryByMount((current) => {
        const next: Record<string, DiskIOSample[]> = {};
        for (const [mount, counters] of Object.entries(diskByMount)) {
          const prevDisk = prev.diskByMount[mount];
          if (!prevDisk) {
            next[mount] = current[mount] ?? [];
            continue;
          }
          const sample: DiskIOSample = {
            readPerSec: ratePerSec(counters.readBytes, prevDisk.readBytes, seconds),
            writePerSec: ratePerSec(counters.writeBytes, prevDisk.writeBytes, seconds),
          };
          next[mount] = [...(current[mount] ?? []), sample].slice(-CHART_HISTORY_MAX_POINTS);
        }
        return next;
      });
    }

    previous.current = {
      at: now,
      network: info.network,
      diskByMount,
    };
  }, [info]);

  return { networkHistory, diskHistoryByMount };
}
