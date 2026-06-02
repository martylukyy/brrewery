import type { ReactNode } from "react";

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

/** Long arc from start to end, sweeping through the top. */
function arcPath(fromDeg: number, toDeg: number) {
  const start = polar(fromDeg);
  const end = polar(toDeg);
  return `M ${start.x} ${start.y} A ${R} ${R} 0 1 1 ${end.x} ${end.y}`;
}

const TRACK_PATH = arcPath(ARC_START, ARC_END);

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

  return (
    <div className="flex h-full min-h-0 flex-col overflow-visible rounded-lg border border-zinc-800 bg-zinc-900/50 p-3">
      <p className="shrink-0 text-center text-xs font-medium uppercase tracking-wide text-zinc-500">
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
              pathLength={100}
              fill="none"
              stroke="currentColor"
              strokeWidth={STROKE}
              strokeLinecap="round"
              className="text-zinc-800"
            />
            {clamped > 0 && (
              <path
                d={TRACK_PATH}
                pathLength={100}
                fill="none"
                stroke="currentColor"
                strokeWidth={STROKE}
                strokeLinecap="round"
                strokeDasharray={`${clamped} 100`}
                className={fillClass}
              />
            )}
          </svg>
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
            <span className="text-2xl font-semibold tabular-nums text-zinc-100">{display}</span>
          </div>
        </div>
      </div>
      <div className="flex min-h-11 w-full flex-1 flex-col items-center justify-center pt-3 text-center px-4 sm:px-8 xl:px-12">
 
        {footer != null && (
          <div className="flex h-full w-full flex-1 flex-col">{footer}</div>
        )}
      </div>
    </div>
  );
}
