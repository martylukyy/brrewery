import { useState } from "react";

import { ChartIntervalSelect } from "@/components/chart-interval-select";
import { ChartPanelControls } from "@/components/chart-panel-controls";
import { ChartScaleSelect } from "@/components/chart-scale-select";
import { ChartPanel } from "@/components/chart-panel";
import { LineChart } from "@/components/line-chart";
import type { NetworkSample } from "@/hooks/use-io-history";
import {
  DEFAULT_CHART_INTERVAL,
  type ChartIntervalId,
  getChartInterval,
  padSeriesRight,
  sliceHistoryForInterval,
} from "@/lib/chart-interval";
import { formatRate } from "@/lib/format";
import {
  DEFAULT_NETWORK_SCALE,
  NETWORK_SCALE_OPTIONS,
  type NetworkScaleId,
  networkScaleMaxBytes,
} from "@/lib/network-scale";

type Props = {
  history: NetworkSample[];
};

export function NetworkThroughputChart({ history }: Props) {
  const [networkScale, setNetworkScale] = useState<NetworkScaleId>(DEFAULT_NETWORK_SCALE);
  const [intervalId, setIntervalId] = useState<ChartIntervalId>(DEFAULT_CHART_INTERVAL);

  const interval = getChartInterval(intervalId);
  const sliced = sliceHistoryForInterval(history, intervalId);
  const pointCount = interval.maxPoints;

  const rx = padSeriesRight(
    sliced.map((s) => s.rxPerSec),
    pointCount,
  );
  const tx = padSeriesRight(
    sliced.map((s) => s.txPerSec),
    pointCount,
  );

  return (
    <ChartPanel
      title="Network throughput"
      waiting={sliced.length < 2}
      pollSeconds={interval.pollMs / 1000}
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
      <LineChart
        pointCount={pointCount}
        series={[
          { label: "Download", colorClass: "text-sky-400", values: rx },
          { label: "Upload", colorClass: "text-emerald-400", values: tx },
        ]}
        maxValue={networkScaleMaxBytes(networkScale)}
        formatValue={formatRate}
      />
    </ChartPanel>
  );
}
