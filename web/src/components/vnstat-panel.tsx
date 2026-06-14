import type { ReactNode } from "react";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { IconArrowDown, IconArrowUp, IconArrowsUpDown } from "@tabler/icons-react";

import { ChartPanelControls } from "@/components/chart-panel-controls";
import { VnstatRangeSelect } from "@/components/vnstat-range-select";
import { Card } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useSetting } from "@/hooks/use-setting";
import { getVnstatReport, type TrafficPeriod } from "@/lib/api";
import { formatBytes } from "@/lib/format";
import { DEFAULT_VNSTAT_RANGE, isVnstatRangeId, type VnstatRangeId } from "@/lib/vnstat-range";

export function VnstatPanel() {
  const vnstat = useQuery({
    queryKey: ["vnstat"],
    queryFn: getVnstatReport,
    refetchInterval: 60_000,
  });
  const [range, setRange] = useSetting<VnstatRangeId>(
    "vnstat-range",
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
        <p className="text-sm text-muted-foreground">Loading vnstat data…</p>
      </Panel>
    );
  }

  if (vnstat.isError) {
    return (
      <Panel title="vnStat - Historic traffic">
        <p className="text-sm text-destructive">{vnstat.error.message}</p>
      </Panel>
    );
  }
  if (!report?.installed) {
    return (
      <Panel title="vnStat - Historic traffic">
        <p className="text-sm text-muted-foreground">
          {report?.message ?? "vnstat is not installed on this system."}
        </p>
        <p className="mt-2 text-xs text-muted-foreground">
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
    <Card className="gap-0 p-4">
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold text-foreground">{title}</h2>
          {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
        </div>
        {headerRight}
      </div>
      {children}
    </Card>
  );
}

function TrafficTable({ periods, reverse = true }: { periods: TrafficPeriod[]; reverse?: boolean }) {
  if (periods.length === 0) {
    return (
      <div>
        <p className="text-sm text-muted-foreground">No data recorded yet.</p>
      </div>
    );
  }

  const rows = reverse ? [...periods].reverse() : periods;

  return (
    <div>
      <div className="overflow-x-auto rounded-md border border-border">
        <Table className="min-w-full text-left text-sm">
          <TableHeader className="bg-muted text-muted-foreground">
            <TableRow>
              <TableHead className="px-3 py-2 font-medium text-foreground">Period</TableHead>
              <TableHead className="px-3 py-2 font-medium text-sky-400">
                <span className="inline-flex items-center gap-1">
                  <IconArrowDown className="size-3.5" aria-hidden="true" />
                  Download
                </span>
              </TableHead>
              <TableHead className="px-3 py-2 font-medium text-emerald-400">
                <span className="inline-flex items-center gap-1">
                  <IconArrowUp className="size-3.5" aria-hidden="true" />
                  Upload
                </span>
              </TableHead>
              <TableHead className="px-3 py-2 font-medium text-orange-400">
                <span className="inline-flex items-center gap-1">
                  <IconArrowsUpDown className="size-3.5" aria-hidden="true" />
                  Total
                </span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.label} className="border-t border-border">
                <TableCell className="px-3 py-2 tabular-nums text-foreground">{row.label}</TableCell>
                <TableCell className="px-3 py-2 tabular-nums text-sky-400">{formatBytes(row.rx_bytes)}</TableCell>
                <TableCell className="px-3 py-2 tabular-nums text-emerald-400">{formatBytes(row.tx_bytes)}</TableCell>
                <TableCell className="px-3 py-2 tabular-nums text-orange-400">{formatBytes(row.rx_bytes + row.tx_bytes)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
