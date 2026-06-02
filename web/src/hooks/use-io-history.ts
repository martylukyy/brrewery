import { useEffect, useRef, useState } from "react";

import type { DiskIOCounters, NetworkCounters, SystemInfo } from "@/lib/api";
import { CHART_HISTORY_MAX_POINTS } from "@/lib/chart-interval";

export type IOSample = {
  rxPerSec: number;
  txPerSec: number;
  readPerSec: number;
  writePerSec: number;
};

type Snapshot = {
  at: number;
  network: NetworkCounters;
  diskIO: DiskIOCounters;
};

function ratePerSec(current: number, previous: number, seconds: number): number {
  if (seconds <= 0 || current < previous) {
    return 0;
  }
  return (current - previous) / seconds;
}

export function useIOHistory(info: SystemInfo | undefined): IOSample[] {
  const previous = useRef<Snapshot | null>(null);
  const [history, setHistory] = useState<IOSample[]>([]);

  useEffect(() => {
    if (!info) {
      return;
    }

    const now = Date.now();
    const prev = previous.current;
    if (prev) {
      const seconds = (now - prev.at) / 1000;
      const sample: IOSample = {
        rxPerSec: ratePerSec(info.network.rx_bytes, prev.network.rx_bytes, seconds),
        txPerSec: ratePerSec(info.network.tx_bytes, prev.network.tx_bytes, seconds),
        readPerSec: ratePerSec(info.disk_io.read_bytes, prev.diskIO.read_bytes, seconds),
        writePerSec: ratePerSec(info.disk_io.write_bytes, prev.diskIO.write_bytes, seconds),
      };
      setHistory((current) => [...current, sample].slice(-CHART_HISTORY_MAX_POINTS));
    }

    previous.current = {
      at: now,
      network: info.network,
      diskIO: info.disk_io,
    };
  }, [info]);

  return history;
}
