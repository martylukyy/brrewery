import { useQuery } from "@tanstack/react-query";

import { DiskIOChart } from "@/components/disk-io-chart";
import { GaugePanel } from "@/components/gauge-panel";
import { NetworkThroughputChart } from "@/components/network-throughput-chart";
import { VnstatPanel } from "@/components/vnstat-panel";
import { useIOHistory } from "@/hooks/use-io-history";
import { getSystemInfo, type LoadAvg } from "@/lib/api";
import { formatBytes, formatUptime } from "@/lib/format";

const SYSTEM_POLL_MS = 1000;

function loadGaugePercent(load1m: number, cpuCount: number): number {
  if (cpuCount <= 0) {
    return 0;
  }
  return Math.min(100, (load1m / cpuCount) * 100);
}

function clampPercent(value: number | undefined): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }
  return Math.min(100, Math.max(0, value));
}

export function Dashboard() {
  const system = useQuery({
    queryKey: ["system"],
    queryFn: getSystemInfo,
    refetchInterval: SYSTEM_POLL_MS,
  });
  const ioHistory = useIOHistory(system.data);

  if (system.isLoading) {
    return <p className="text-zinc-400">Loading system metrics…</p>;
  }

  if (system.isError) {
    return <p className="text-red-400">{system.error.message}</p>;
  }

  const info = system.data;
  if (!info) {
    return <p className="text-zinc-400">No system metrics available.</p>;
  }

  const memoryPercent = Math.min(100, Math.max(0, info.memory.used_percent));
  const diskPercent = Math.min(100, Math.max(0, info.disk.used_percent));
  const cpuPercent = Math.min(100, Math.max(0, info.cpu_percent));
  const diskIOBusyPercent = clampPercent(info.disk_io_busy_percent);
  const loadPercent = loadGaugePercent(info.load["1m"], info.cpu_count);

  return (
    <div className="space-y-6">
      <div className="flex items-baseline justify-between gap-4">
        <h1 className="text-2xl font-semibold text-zinc-100">brrewery</h1>
        <p className="text-sm text-zinc-400">
          {info.hostname} · uptime {formatUptime(info.uptime_seconds)}
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4 [&>*]:h-full">
        <GaugePanel
          label="CPU"
          value={cpuPercent}
          display={`${cpuPercent.toFixed(1)}%`}
          footer={
            <p className="line-clamp-2 flex h-full items-center justify-center text-center text-xs text-zinc-500">{info.cpu_name}</p>
          }
        />
        <LoadGaugePanel load={info.load} gaugePercent={loadPercent} />
        <GaugePanel
          label="Memory"
          value={memoryPercent}
          display={`${memoryPercent.toFixed(1)}%`}
          footer={
            <p className="flex h-full items-center justify-center text-center text-xs text-zinc-500">
              {formatBytes(info.memory.used_bytes)} / {formatBytes(info.memory.total_bytes)}
            </p>
          }
        />

      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4 [&>*]:h-full">
        <GaugePanel
          label={`Disk (${info.disk.mount})`}
          value={diskPercent}
          display={`${diskPercent.toFixed(1)}%`}
          footer={
            <p className="flex h-full items-center justify-center text-center text-xs text-zinc-500">
              {formatBytes(info.disk.used_bytes)} / {formatBytes(info.disk.total_bytes)}
            </p>
          }
        />
        <GaugePanel
          label="I/O busy"
          value={diskIOBusyPercent}
          display={`${diskIOBusyPercent.toFixed(2)}%`}
        />
        <div className="md:col-span-2 xl:col-span-2">
          <DiskIOChart history={ioHistory} />
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <NetworkThroughputChart history={ioHistory} />
        <VnstatPanel />
      </div>
    </div>
  );
}

function LoadGaugePanel({ load, gaugePercent }: { load: LoadAvg; gaugePercent: number }) {
  const windows = [
    { label: "1m", value: load["1m"] },
    { label: "5m", value: load["5m"] },
    { label: "15m", value: load["15m"] },
  ] as const;

  return (
    <GaugePanel
      label="Load average"
      value={gaugePercent}
      display={load["1m"].toFixed(2)}
      footer={
        <div className="flex h-full w-full items-center justify-between">
          {windows.map((slot) => (
            <div key={slot.label} className="flex flex-col items-center">
              <span className="text-xs font-medium text-zinc-500">{slot.label}</span>
              <span className="mt-0.5 text-sm font-semibold tabular-nums text-zinc-200">
                {slot.value.toFixed(2)}
              </span>
            </div>
          ))}
        </div>
      }
    />
  );
}
