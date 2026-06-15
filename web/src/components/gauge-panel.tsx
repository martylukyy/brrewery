import type { ReactNode } from "react";

import { Card } from "@/components/ui/card";
import {
  GAUGE_SLOT_HEIGHT_PX,
  STROKE,
  TRACK_PATH,
  VIEW_BOX,
  gaugeFillClass,
  gaugeFillPath,
} from "@/lib/gauge";

type Props = {
  label: string;
  /** 0–100 fill level for the gauge arc. */
  value: number;
  /** Center readout (e.g. `42.5%` or `0.50`). */
  display: string;
  footer?: ReactNode;
};

export function GaugePanel({ label, value, display, footer }: Props) {
  const clamped = Math.min(100, Math.max(0, value));
  const fillClass = gaugeFillClass(clamped);
  const fillPath = gaugeFillPath(clamped);

  return (
    <Card className="flex h-full min-h-0 flex-col gap-0 overflow-visible p-3 py-3">
      <p className="shrink-0 text-center text-xs font-medium uppercase tracking-wide">
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
