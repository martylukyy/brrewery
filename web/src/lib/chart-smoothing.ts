/** Window of the trailing moving average applied to throughput series. */
export const SMOOTHING_WINDOW_SECONDS = 5;

/**
 * Smooths a series with a trailing moving average spanning
 * `SMOOTHING_WINDOW_SECONDS` of samples at the given poll cadence.
 * The leading points average over however many samples exist so far.
 */
export function smoothSeries(values: number[], pollMs: number): number[] {
  const windowSize = Math.max(1, Math.round((SMOOTHING_WINDOW_SECONDS * 1000) / pollMs));
  if (windowSize === 1) {
    return values;
  }

  const smoothed: number[] = [];
  let sum = 0;
  for (let i = 0; i < values.length; i++) {
    sum += values[i];
    if (i >= windowSize) {
      sum -= values[i - windowSize];
    }
    smoothed.push(sum / Math.min(i + 1, windowSize));
  }
  return smoothed;
}
