import { memo, useMemo } from "react";

import { ChartIntervalSelect } from "@/components/chart-interval-select";
import { ChartPanelControls } from "@/components/chart-panel-controls";
import { ChartPanel } from "@/components/chart-panel";
import { LineChart } from "@/components/line-chart";
import { useChartWidth } from "@/hooks/use-chart-width";
import { useIOHistory } from "@/hooks/use-io-history";
import { useSetting } from "@/hooks/use-setting";
import {
  CHART_SAMPLE_SECONDS,
  DEFAULT_CHART_INTERVAL,
  type ChartIntervalId,
  isChartIntervalId,
} from "@/lib/chart-interval";
import { formatRate } from "@/lib/format";

type Props = {
  chartIdSuffix: string;
  mountPoint: string;
};

export const DiskIOChart = memo(function DiskIOChart({ chartIdSuffix, mountPoint }: Props) {
  const [intervalId, setIntervalId] = useSetting<ChartIntervalId>(
    `disk-chart-interval:${chartIdSuffix}`,
    DEFAULT_CHART_INTERVAL,
    isChartIntervalId,
  );

  const { ref, width } = useChartWidth();
  const { seriesByKey, pointCount, hasSamples } = useIOHistory(intervalId, width, mountPoint);

  const series = useMemo(
    () => [
      { label: "Read", color: "var(--color-sky-400)", values: seriesByKey.read ?? [] },
      { label: "Write", color: "var(--color-emerald-400)", values: seriesByKey.write ?? [] },
    ],
    [seriesByKey],
  );

  return (
    <ChartPanel
      title={`Disk throughput ( ${mountPoint} )`}
      waiting={!hasSamples}
      pollSeconds={CHART_SAMPLE_SECONDS}
      action={
        <ChartPanelControls
          timeRange={
            <ChartIntervalSelect
              id={`disk-chart-interval-${chartIdSuffix}`}
              value={intervalId}
              onChange={setIntervalId}
            />
          }
        />
      }
    >
      <div ref={ref} className="flex min-h-0 flex-1 flex-col">
        <LineChart pointCount={pointCount} series={series} formatValue={formatRate} />
      </div>
    </ChartPanel>
  );
});
