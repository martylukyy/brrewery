import { useState } from "react";

import { ChartIntervalSelect } from "@/components/chart-interval-select";
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
        <div className="flex flex-wrap items-center justify-end gap-3">
          <ChartIntervalSelect
            id="network-chart-interval"
            value={intervalId}
            onChange={setIntervalId}
          />
          <label className="flex items-center gap-2 text-xs text-zinc-500">
            <span>Scale</span>
            <select
              value={networkScale}
              onChange={(e) => setNetworkScale(e.target.value as NetworkScaleId)}
              className="rounded-md border border-zinc-700 bg-zinc-950 px-2 py-1 text-xs text-zinc-200"
              aria-label="Network chart scale"
            >
              {NETWORK_SCALE_OPTIONS.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
        </div>
      }
    >
      <LineChart
        pointCount={pointCount}
        series={[
          { label: "Download", color: "#38bdf8", values: rx },
          { label: "Upload", color: "#a78bfa", values: tx },
        ]}
        maxValue={networkScaleMaxBytes(networkScale)}
        formatValue={formatRate}
      />
    </ChartPanel>
  );
}
