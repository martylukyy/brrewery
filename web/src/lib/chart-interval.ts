export const CHART_INTERVAL_OPTIONS = [
  { id: "1m", label: "1 minute", pollMs: 1000, maxPoints: 60 },
  { id: "5m", label: "5 minutes", pollMs: 1000, maxPoints: 300 },
  { id: "15m", label: "15 minutes", pollMs: 1000, maxPoints: 900 },
  { id: "30m", label: "30 minutes", pollMs: 1000, maxPoints: 1800 },
  { id: "1h", label: "1 hour", pollMs: 1000, maxPoints: 3600 },
  { id: "2h", label: "2 hours", pollMs: 1000, maxPoints: 7200 },
  { id: "4h", label: "4 hours", pollMs: 1000, maxPoints: 14_400 },
  { id: "8h", label: "8 hours", pollMs: 1000, maxPoints: 28_800 },
  { id: "12h", label: "12 hours", pollMs: 1000, maxPoints: 43_200 },
  { id: "24h", label: "24 hours", pollMs: 1000, maxPoints: 86_400 },
] as const;

export type ChartIntervalId = (typeof CHART_INTERVAL_OPTIONS)[number]["id"];

export const DEFAULT_CHART_INTERVAL: ChartIntervalId = "5m";

export const CHART_HISTORY_MAX_POINTS = CHART_INTERVAL_OPTIONS.reduce(
  (max, option) => Math.max(max, option.maxPoints),
  0,
);

export function getChartInterval(id: ChartIntervalId) {
  const option = CHART_INTERVAL_OPTIONS.find((entry) => entry.id === id);
  return option ?? CHART_INTERVAL_OPTIONS[1];
}

export function isChartIntervalId(value: unknown): value is ChartIntervalId {
  return CHART_INTERVAL_OPTIONS.some((entry) => entry.id === value);
}

export function sliceHistoryForInterval<T>(history: T[], intervalId: ChartIntervalId): T[] {
  return history.slice(-getChartInterval(intervalId).maxPoints);
}

/** Right-aligns values in a fixed-length series; leading slots are empty. */
export function padSeriesRight(values: number[], pointCount: number): (number | null)[] {
  if (pointCount <= 0) {
    return [];
  }

  const padded: (number | null)[] = Array.from({ length: pointCount }, () => null);
  const start = Math.max(0, pointCount - values.length);
  for (let i = 0; i < values.length; i++) {
    padded[start + i] = values[i] ?? null;
  }
  return padded;
}
