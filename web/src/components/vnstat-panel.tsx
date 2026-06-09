import type { ReactNode } from "react";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";

import { ChartPanelControls } from "@/components/chart-panel-controls";
import { VnstatRangeSelect } from "@/components/vnstat-range-select";
import { useLocalStorageState } from "@/hooks/use-local-storage-state";
import { getVnstatReport, type TrafficPeriod } from "@/lib/api";
import { formatBytes } from "@/lib/format";
import { DEFAULT_VNSTAT_RANGE, isVnstatRangeId, type VnstatRangeId } from "@/lib/vnstat-range";

export function VnstatPanel() {
  const vnstat = useQuery({
    queryKey: ["vnstat"],
    queryFn: getVnstatReport,
    refetchInterval: 60_000,
  });
  const [range, setRange] = useLocalStorageState<VnstatRangeId>(
    "brrewery:vnstat-range",
    DEFAULT_VNSTAT_RANGE,
    isVnstatRangeId,
  );
  const report = vnstat.data;
  const tableConfig = useMemo(() => {
    switch (range) {
      case "months":
        return {
          periods: report?.months ?? [],
        };
      case "top10":
        return {
          periods: [...(report?.days ?? [])]
            .sort((a, b) => b.rx_bytes + b.tx_bytes - (a.rx_bytes + a.tx_bytes))
            .slice(0, 10),
        };
      case "days":
      default:
        return {
          periods: report?.days ?? [],
        };
    }
  }, [range, report?.days, report?.months]);

  if (vnstat.isLoading) {
    return (
      <Panel title="vnStat - Historic traffic">
        <p className="text-sm text-zinc-400">Loading vnstat data…</p>
      </Panel>
    );
  }

  if (vnstat.isError) {
    return (
      <Panel title="vnStat - Historic traffic">
        <p className="text-sm text-red-400">{vnstat.error.message}</p>
      </Panel>
    );
  }
  if (!report?.installed) {
    return (
      <Panel title="vnStat - Historic traffic">
        <p className="text-sm text-zinc-400">
          {report?.message ?? "vnstat is not installed on this system."}
        </p>
        <p className="mt-2 text-xs text-zinc-600">
          Install vnstat and ensure the daemon is collecting data for your interfaces.
        </p>
      </Panel>
    );
  }

  return (
    <Panel
      title="vnStat - Historic traffic"
      headerRight={
        <ChartPanelControls
          timeRange={<VnstatRangeSelect value={range} onChange={setRange} />}
        />
      }
    >
      <div>
        <TrafficTable periods={tableConfig.periods} reverse={range !== "top10"} />
      </div>
    </Panel>
  );
}

function Panel({
  title,
  subtitle,
  headerRight,
  children,
}: {
  title: string;
  subtitle?: string;
  headerRight?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 p-4">
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold text-zinc-200">{title}</h2>
          {subtitle && <p className="text-xs text-zinc-500">{subtitle}</p>}
        </div>
        {headerRight}
      </div>
      {children}
    </div>
  );
}

function TrafficTable({ periods, reverse = true }: { periods: TrafficPeriod[]; reverse?: boolean }) {
  if (periods.length === 0) {
    return (
      <div>
        <p className="text-sm text-zinc-500">No data recorded yet.</p>
      </div>
    );
  }

  const rows = reverse ? [...periods].reverse() : periods;

  return (
    <div>
      <div className="overflow-x-auto rounded-md border border-zinc-800">
        <table className="min-w-full text-left text-sm">
          <thead className="bg-zinc-900 text-zinc-500">
            <tr>
              <th className="px-3 py-2 font-medium text-zinc-100">Period</th>
              <th className="px-3 py-2 font-medium text-sky-400">Download</th>
              <th className="px-3 py-2 font-medium text-emerald-400">Upload</th>
              <th className="px-3 py-2 font-medium text-orange-400">Total</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.label} className="border-t border-zinc-800">
                <td className="px-3 py-2 tabular-nums text-zinc-100">{row.label}</td>
                <td className="px-3 py-2 tabular-nums text-sky-400">{formatBytes(row.rx_bytes)}</td>
                <td className="px-3 py-2 tabular-nums text-emerald-400">{formatBytes(row.tx_bytes)}</td>
                <td className="px-3 py-2 tabular-nums text-orange-400">{formatBytes(row.rx_bytes + row.tx_bytes)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
