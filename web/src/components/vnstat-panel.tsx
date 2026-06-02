import type { ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";

import { getVnstatReport, type TrafficPeriod } from "@/lib/api";
import { formatBytes } from "@/lib/format";

export function VnstatPanel() {
  const vnstat = useQuery({
    queryKey: ["vnstat"],
    queryFn: getVnstatReport,
    refetchInterval: 60_000,
  });

  if (vnstat.isLoading) {
    return (
      <Panel title="Historic traffic (vnstat)">
        <p className="text-sm text-zinc-400">Loading vnstat data…</p>
      </Panel>
    );
  }

  if (vnstat.isError) {
    return (
      <Panel title="Historic traffic (vnstat)">
        <p className="text-sm text-red-400">{vnstat.error.message}</p>
      </Panel>
    );
  }

  const report = vnstat.data;
  if (!report?.installed) {
    return (
      <Panel title="Historic traffic (vnstat)">
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
      title="Historic traffic (vnstat)"
      subtitle={report.version ? `vnstat ${report.version}` : undefined}
    >
      <div className="grid gap-6 lg:grid-cols-2">
        <TrafficTable title="Last 12 months" periods={report.months ?? []} />
        <TrafficTable title="Last 30 days" periods={report.days ?? []} />
      </div>
    </Panel>
  );
}

function Panel({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children: ReactNode;
}) {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 p-4">
      <div className="mb-4">
        <h2 className="text-sm font-semibold text-zinc-200">{title}</h2>
        {subtitle && <p className="text-xs text-zinc-500">{subtitle}</p>}
      </div>
      {children}
    </div>
  );
}

function TrafficTable({ title, periods }: { title: string; periods: TrafficPeriod[] }) {
  if (periods.length === 0) {
    return (
      <div>
        <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-zinc-500">{title}</h3>
        <p className="text-sm text-zinc-500">No data recorded yet.</p>
      </div>
    );
  }

  const rows = [...periods].reverse();

  return (
    <div>
      <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-zinc-500">{title}</h3>
      <div className="overflow-x-auto rounded-md border border-zinc-800">
        <table className="min-w-full text-left text-sm">
          <thead className="bg-zinc-900 text-zinc-500">
            <tr>
              <th className="px-3 py-2 font-medium">Period</th>
              <th className="px-3 py-2 font-medium">Download</th>
              <th className="px-3 py-2 font-medium">Upload</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.label} className="border-t border-zinc-800">
                <td className="px-3 py-2 tabular-nums text-zinc-300">{row.label}</td>
                <td className="px-3 py-2 tabular-nums text-zinc-400">{formatBytes(row.rx_bytes)}</td>
                <td className="px-3 py-2 tabular-nums text-zinc-400">{formatBytes(row.tx_bytes)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
