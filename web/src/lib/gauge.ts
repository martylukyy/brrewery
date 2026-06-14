// Pure SVG geometry + fill helpers for the dashboard gauges. Kept out of the
// component module so the math can be unit-tested in isolation and so
// gauge-panel.tsx only exports a component (Fast Refresh friendly).

/** Arc stroke color from fill level: 0–80 green, 80–90 orange, 90–100 red. */
export function gaugeFillClass(value: number): string {
  if (value > 90) {
    return "text-red-700";
  }
  if (value > 80) {
    return "text-orange-500";
  }
  return "text-emerald-500";
}

const CX = 50;
const CY = 64;
const R = 42;
export const STROKE = 6;
const PAD = STROKE / 2 + 3;
/** Shared gauge drawing area height so arc endpoints align across panels. */
export const GAUGE_SLOT_HEIGHT_PX = 190;
/** 300° arc symmetric about the vertical axis (endpoints share the same Y). */
const ARC_DEGREES = 300;
const ARC_HALF = ARC_DEGREES / 2;
const ARC_START = 90 + ARC_HALF;
const ARC_END = 90 - ARC_HALF;

function polar(angleDeg: number) {
  const rad = (angleDeg * Math.PI) / 180;
  return {
    x: CX + R * Math.cos(rad),
    y: CY - R * Math.sin(rad),
  };
}

function arcSweepSpan(fromDeg: number, toDeg: number): number {
  let span = fromDeg - toDeg;
  while (span <= 0) {
    span += 360;
  }
  return span;
}

function arcPath(fromDeg: number, toDeg: number) {
  const start = polar(fromDeg);
  const end = polar(toDeg);
  const largeArc = arcSweepSpan(fromDeg, toDeg) > 180 ? 1 : 0;
  return `M ${start.x} ${start.y} A ${R} ${R} 0 ${largeArc} 1 ${end.x} ${end.y}`;
}

export const TRACK_PATH = arcPath(ARC_START, ARC_END);

/** Partial fill arc from the left endpoint toward the right, through the top. */
export function gaugeFillPath(percent: number): string | null {
  const clamped = Math.min(100, Math.max(0, percent));
  if (clamped <= 0) {
    return null;
  }
  if (clamped >= 100) {
    return TRACK_PATH;
  }
  const fillEnd = ARC_START - (clamped / 100) * ARC_DEGREES;
  return arcPath(ARC_START, fillEnd);
}

function gaugeViewBox() {
  const sampleAngles = [ARC_START, ARC_END, 90, 180, 0];
  let minX = CX - R;
  let maxX = CX + R;
  let minY = Infinity;
  let maxY = -Infinity;

  for (const angle of sampleAngles) {
    const point = polar(angle);
    minX = Math.min(minX, point.x);
    maxX = Math.max(maxX, point.x);
    minY = Math.min(minY, point.y);
    maxY = Math.max(maxY, point.y);
  }

  return {
    minX: minX - PAD,
    minY: minY - PAD,
    width: maxX - minX + PAD * 2,
    height: maxY - minY + PAD * 2,
  };
}

export const VIEW_BOX = gaugeViewBox();
