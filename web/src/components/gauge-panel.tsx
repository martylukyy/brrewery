import type { ReactNode } from "react";

import { Card } from "@/components/ui/card";

type Props = {
  label: string;
  /** 0–100 fill level for the gauge arc. */
  value: number;
  /** Center readout (e.g. `42.5%` or `0.50`). */
  display: string;
  footer?: ReactNode;
};

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
const STROKE = 6;
const PAD = STROKE / 2 + 3;
/** Shared gauge drawing area height so arc endpoints align across panels. */
const GAUGE_SLOT_HEIGHT_PX = 190;
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

const TRACK_PATH = arcPath(ARC_START, ARC_END);

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

const VIEW_BOX = gaugeViewBox();

export function GaugePanel({ label, value, display, footer }: Props) {
  const clamped = Math.min(100, Math.max(0, value));
  const fillClass = gaugeFillClass(clamped);
  const fillPath = gaugeFillPath(clamped);

  return (
    <Card className="flex h-full min-h-0 flex-col gap-0 overflow-visible p-3 py-3">
      <p className="shrink-0 text-center text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </p>
      <div
        className="mx-auto mt-3 flex w-full min-w-0 shrink-0 items-end justify-center overflow-visible"
        style={{ height: GAUGE_SLOT_HEIGHT_PX }}
      >
        <div
          className="relative h-full w-[18rem] min-w-[18rem] max-w-none"
          role="img"
          aria-label={`${label}: ${display}`}
        >
          <svg
            viewBox={`${VIEW_BOX.minX} ${VIEW_BOX.minY} ${VIEW_BOX.width} ${VIEW_BOX.height}`}
            className="h-full w-full"
            preserveAspectRatio="xMidYMax meet"
            aria-hidden
          >
            <path
              d={TRACK_PATH}
              fill="none"
              stroke="currentColor"
              strokeWidth={STROKE}
              strokeLinecap="round"
              className="text-border"
            />
            {fillPath && (
              <path
                d={fillPath}
                fill="none"
                stroke="currentColor"
                strokeWidth={STROKE}
                strokeLinecap="round"
                className={fillClass}
              />
            )}
          </svg>
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
            <span className="text-2xl font-semibold tabular-nums text-foreground">{display}</span>
          </div>
        </div>
      </div>
      <div className="flex min-h-11 w-full flex-1 flex-col items-center justify-center pt-3 text-center px-4 sm:px-8 xl:px-12">
 
        {footer != null && (
          <div className="flex h-full w-full flex-1 flex-col">{footer}</div>
        )}
      </div>
    </Card>
  );
}
