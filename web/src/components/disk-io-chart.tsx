import { useState } from "react";

import { ChartIntervalSelect } from "@/components/chart-interval-select";
import { ChartPanel } from "@/components/chart-panel";
import { LineChart } from "@/components/line-chart";
import type { DiskIOSample } from "@/hooks/use-io-history";
import {
  DEFAULT_CHART_INTERVAL,
  type ChartIntervalId,
  getChartInterval,
  padSeriesRight,
  sliceHistoryForInterval,
} from "@/lib/chart-interval";
import { formatRate } from "@/lib/format";

type Props = {
  history: DiskIOSample[];
  chartIdSuffix: string;
  mountPoint: string;
};

export function DiskIOChart({ history, chartIdSuffix, mountPoint }: Props) {
  const [intervalId, setIntervalId] = useState<ChartIntervalId>(DEFAULT_CHART_INTERVAL);

  const interval = getChartInterval(intervalId);
  const sliced = sliceHistoryForInterval(history, intervalId);
  const pointCount = interval.maxPoints;

  const read = padSeriesRight(
    sliced.map((s) => s.readPerSec),
    pointCount,
  );
  const write = padSeriesRight(
    sliced.map((s) => s.writePerSec),
    pointCount,
  );

  return (
    <ChartPanel
      title={`${mountPoint} throughput`}
      waiting={sliced.length < 2}
      pollSeconds={interval.pollMs / 1000}
      action={
        <ChartIntervalSelect
          id={`disk-chart-interval-${chartIdSuffix}`}
          value={intervalId}
          onChange={setIntervalId}
        />
      }
    >
      <LineChart
        pointCount={pointCount}
        series={[
          { label: "Read", color: "#34d399", values: read },
          { label: "Write", color: "#fbbf24", values: write },
        ]}
        formatValue={formatRate}
      />
    </ChartPanel>
  );
}
