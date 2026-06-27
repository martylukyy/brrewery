// Gauge status helpers. Kept out of the component module so the thresholds can
// be unit-tested in isolation and so gauge-panel.tsx only exports a component
// (Fast Refresh friendly).

/** Arc color from fill level: 0–80 ok (green), 80–90 warn (orange), 90–100 crit (red). */
export function gaugeColor(value: number): string {
  if (value > 90) {
    return "var(--gauge-crit)";
  }
  if (value > 80) {
    return "var(--gauge-warn)";
  }
  return "var(--gauge-ok)";
}

/** Shared gauge drawing area height so the arcs align across panels. */
export const GAUGE_SLOT_HEIGHT_PX = 190;
