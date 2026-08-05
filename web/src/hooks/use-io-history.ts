import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import { getIOHistory, type IOHistorySeries } from "@/lib/api";
import {
  CHART_SAMPLE_SECONDS,
  chartPointBudget,
  type ChartIntervalId,
} from "@/lib/chart-interval";

const REFETCH_MS = CHART_SAMPLE_SECONDS * 1000;

export type IOChartData = {
  /** One entry per requested bucket; null where no sample was recorded. */
  seriesByKey: Record<string, (number | null)[]>;
  pointCount: number;
  hasSamples: boolean;
  error: Error | null;
};

const EMPTY_SERIES: Record<string, (number | null)[]> = {};

function indexSeries(series: IOHistorySeries[]): Record<string, (number | null)[]> {
  return Object.fromEntries(series.map((entry) => [entry.key, entry.points]));
}

/**
 * Loads the daemon-side throughput history for one chart.
 *
 * The daemon retains 24h of one-second samples and does the smoothing and
 * downsampling, so the browser holds nothing but the points it draws: a range
 * change is a request, not a wait for history to accumulate, and a backgrounded
 * tab misses nothing.
 *
 * `mount` selects a disk; omit it for network throughput.
 */
export function useIOHistory(
  intervalId: ChartIntervalId,
  widthPx: number,
  mount?: string,
): IOChartData {
  const points = chartPointBudget(intervalId, widthPx);

  const query = useQuery({
    queryKey: ["io-history", mount ?? "network", intervalId, points],
    queryFn: () => getIOHistory({ range: intervalId, points, mount }),
    refetchInterval: REFETCH_MS,
    // Keep the previous range's points on screen while the new range loads,
    // instead of blanking the chart on every range change.
    placeholderData: (previous) => previous,
  });

  // Derived once per response rather than per render: the dashboard re-renders
  // every second and these arrays are as long as the chart is wide.
  const data = useMemo(() => {
    const seriesByKey = query.data ? indexSeries(query.data.series ?? []) : EMPTY_SERIES;
    const values = Object.values(seriesByKey);
    return {
      seriesByKey,
      // The daemon clamps the bucket count to the samples the range can hold,
      // so trust its answer over the requested budget.
      pointCount: values[0]?.length ?? points,
      hasSamples: values.some((series) => series.some((value) => value != null)),
    };
  }, [query.data, points]);

  return { ...data, error: query.error };
}
