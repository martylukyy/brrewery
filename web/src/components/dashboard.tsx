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
    // Keep sampling while the browser tab is backgrounded so the throughput
    // history keeps filling instead of leaving a gap until the tab is reopened.
    refetchIntervalInBackground: true,
  });
  const { networkHistory, diskHistoryByMount } = useIOHistory(system.data);

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
  const cpuPercent = Math.min(100, Math.max(0, info.cpu_percent));
  const disks = info.disks ?? [];
  const loadPercent = loadGaugePercent(info.load["1m"], info.cpu_count);
  const showDiskHeadings = disks.length > 1;

  return (
    <div className="space-y-6">
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
        <UptimePanel uptimeSeconds={info.uptime_seconds} hostname={info.hostname} />
      </div>

      {disks.map((disk) => {
        const usedPercent = Math.min(100, Math.max(0, disk.used_percent));
        const busyPercent = clampPercent(disk.io_busy_percent);
        const chartIdSuffix = disk.mount.replaceAll("/", "-").replaceAll(" ", "-");
        return (
          <section key={disk.mount} className="space-y-3">
            {showDiskHeadings ? (
              <h2 className="text-lg font-semibold text-zinc-100">{disk.mount}</h2>
            ) : null}
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4 [&>*]:h-full">
              <GaugePanel
                label="Disk usage"
                value={usedPercent}
                display={`${usedPercent.toFixed(1)}%`}
                footer={
                  <p className="flex h-full items-center justify-center text-center text-xs text-zinc-500">
                    {formatBytes(disk.used_bytes)} / {formatBytes(disk.total_bytes)}
                  </p>
                }
              />
              <GaugePanel
                label="I/O busy"
                value={busyPercent}
                display={`${busyPercent.toFixed(2)}%`}
              />
              <div className="md:col-span-2 xl:col-span-2">
                <DiskIOChart
                  history={diskHistoryByMount[disk.mount] ?? []}
                  chartIdSuffix={chartIdSuffix}
                  mountPoint={disk.mount}
                />
              </div>
            </div>
          </section>
        );
      })}

      <div className="grid gap-4 lg:grid-cols-2">
        <NetworkThroughputChart history={networkHistory} />
        <VnstatPanel />
      </div>
    </div>
  );
}

function UptimePanel({ uptimeSeconds, hostname }: { uptimeSeconds: number; hostname: string }) {
  return (
    <div className="flex h-full min-h-0 flex-col rounded-lg border border-zinc-800 bg-zinc-900/50 p-3">
      <p className="shrink-0 text-center text-xs font-medium uppercase tracking-wide text-zinc-500">
        Uptime
      </p>
      <div className="flex flex-1 flex-col items-center justify-center py-6">
        <span className="text-2xl font-semibold tabular-nums text-zinc-100">
          {formatUptime(uptimeSeconds)}
        </span>
      </div>
      <div className="flex min-h-11 items-center justify-center px-4 text-center">
        <p className="line-clamp-2 text-xs text-zinc-500">{hostname}</p>
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
