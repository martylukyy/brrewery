// Chart ranges. Each id is also the `range` query parameter the daemon parses
// as a Go duration, and every range is served from the same 24h of history the
// daemon retains.
export const CHART_INTERVAL_OPTIONS = [
  { id: "1m", label: "1 minute", seconds: 60 },
  { id: "5m", label: "5 minutes", seconds: 300 },
  { id: "15m", label: "15 minutes", seconds: 900 },
  { id: "30m", label: "30 minutes", seconds: 1800 },
  { id: "1h", label: "1 hour", seconds: 3600 },
  { id: "2h", label: "2 hours", seconds: 7200 },
  { id: "4h", label: "4 hours", seconds: 14_400 },
  { id: "8h", label: "8 hours", seconds: 28_800 },
  { id: "12h", label: "12 hours", seconds: 43_200 },
  { id: "24h", label: "24 hours", seconds: 86_400 },
] as const;

export type ChartIntervalId = (typeof CHART_INTERVAL_OPTIONS)[number]["id"];

export const DEFAULT_CHART_INTERVAL: ChartIntervalId = "5m";

/** The daemon samples I/O counters at this rate, whatever range is displayed. */
export const CHART_SAMPLE_SECONDS = 1;

/** Matches the daemon's per-request bucket cap. */
const MAX_CHART_POINTS = 4000;

export function getChartInterval(id: ChartIntervalId) {
  const option = CHART_INTERVAL_OPTIONS.find((entry) => entry.id === id);
  return option ?? CHART_INTERVAL_OPTIONS[1];
}

export function isChartIntervalId(value: unknown): value is ChartIntervalId {
  return CHART_INTERVAL_OPTIONS.some((entry) => entry.id === value);
}

/**
 * Number of points to request for a range drawn `widthPx` pixels wide. Drawing
 * more points than the plot has pixels costs work nobody can see — a 24h range
 * holds 86 400 samples for a plot a few hundred pixels wide — so the daemon
 * downsamples to this budget instead of sending the raw series. It is also
 * never finer than the sample rate, which would only yield empty buckets.
 */
export function chartPointBudget(intervalId: ChartIntervalId, widthPx: number): number {
  const samples = Math.floor(getChartInterval(intervalId).seconds / CHART_SAMPLE_SECONDS);
  return Math.max(1, Math.min(Math.floor(widthPx), samples, MAX_CHART_POINTS));
}
