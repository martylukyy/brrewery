import { memo, useMemo } from "react";

import { ChartIntervalSelect } from "@/components/chart-interval-select";
import { ChartPanelControls } from "@/components/chart-panel-controls";
import { ChartScaleSelect } from "@/components/chart-scale-select";
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
import {
  DEFAULT_NETWORK_SCALE,
  NETWORK_SCALE_OPTIONS,
  type NetworkScaleId,
  isNetworkScaleId,
  networkScaleMaxBytes,
} from "@/lib/network-scale";

export const NetworkThroughputChart = memo(function NetworkThroughputChart() {
  const [networkScale, setNetworkScale] = useSetting<NetworkScaleId>(
    "network-chart-scale",
    DEFAULT_NETWORK_SCALE,
    isNetworkScaleId,
  );
  const [intervalId, setIntervalId] = useSetting<ChartIntervalId>(
    "network-chart-interval",
    DEFAULT_CHART_INTERVAL,
    isChartIntervalId,
  );

  const { ref, width } = useChartWidth();
  const { seriesByKey, pointCount, hasSamples } = useIOHistory(intervalId, width);

  const series = useMemo(
    () => [
      { label: "Download", color: "var(--color-sky-400)", values: seriesByKey.rx ?? [] },
      { label: "Upload", color: "var(--color-emerald-400)", values: seriesByKey.tx ?? [] },
    ],
    [seriesByKey],
  );

  return (
    <ChartPanel
      title="Network throughput"
      waiting={!hasSamples}
      pollSeconds={CHART_SAMPLE_SECONDS}
      action={
        <ChartPanelControls
          leading={
            <ChartScaleSelect
              id="network-chart-scale"
              value={networkScale}
              options={NETWORK_SCALE_OPTIONS}
              onChange={setNetworkScale}
              ariaLabel="Network chart scale"
            />
          }
          timeRange={
            <ChartIntervalSelect
              id="network-chart-interval"
              value={intervalId}
              onChange={setIntervalId}
            />
          }
        />
      }
    >
      <div ref={ref} className="flex min-h-0 flex-1 flex-col">
        <LineChart
          pointCount={pointCount}
          series={series}
          maxValue={networkScaleMaxBytes(networkScale)}
          formatValue={formatRate}
        />
      </div>
    </ChartPanel>
  );
});
