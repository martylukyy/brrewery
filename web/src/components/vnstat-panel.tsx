import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { getVnstatReport, type TrafficPeriod } from "@/lib/api";
import { formatBytes } from "@/lib/format";

export function VnstatPanel() {
  const vnstat = useQuery({
    queryKey: ["vnstat"],
    queryFn: getVnstatReport,
    refetchInterval: 60_000,
  });
  const [range, setRange] = useState<"days" | "months" | "top10">("days");
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
        <label className="flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-zinc-500">
          Range
          <select
            className="w-fit rounded-md border border-zinc-700 bg-zinc-900 px-2 py-1 text-sm normal-case tracking-normal text-zinc-200 focus:border-zinc-500 focus:outline-none"
            value={range}
            onChange={(event) => {
              setRange(event.target.value as "days" | "months" | "top10");
            }}
          >
            <option value="months">Last 12 months</option>
            <option value="days">Last 30 days</option>
            <option value="top10">Top 10 days overall</option>
          </select>
        </label>
      }
    >
      <div>
        <TrafficTable periods={tableConfig.periods} />
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

function TrafficTable({ periods }: { periods: TrafficPeriod[] }) {
  if (periods.length === 0) {
    return (
      <div>
        <p className="text-sm text-zinc-500">No data recorded yet.</p>
      </div>
    );
  }

  const rows = [...periods].reverse();

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
