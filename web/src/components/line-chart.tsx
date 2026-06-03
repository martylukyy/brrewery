export type LineSeries = {
  label: string;
  /** Tailwind text color class for the line and legend dot (e.g. text-sky-400). */
  colorClass: string;
  values: (number | null)[];
};

type Props = {
  series: LineSeries[];
  height?: number;
  /** Fixed number of X-axis slots (e.g. time-range sample count). */
  pointCount?: number;
  /** Fixed Y-axis maximum in the same units as series values. When unset, scales to data. */
  maxValue?: number;
  formatValue?: (value: number) => string;
};

const PLOT_WIDTH = 436;
const PAD_RIGHT = 4;
const PAD_TOP = 4;
const PAD_BOTTOM = 4;
const Y_TICKS = [1, 0.5, 0] as const;

function plotHeight(height: number): number {
  return height - PAD_TOP - PAD_BOTTOM;
}

function plotInnerWidth(): number {
  return PLOT_WIDTH - PAD_RIGHT;
}

function valueY(value: number, height: number, max: number): number {
  const clamped = Math.min(Math.max(value, 0), max);
  return height - PAD_BOTTOM - (clamped / max) * plotHeight(height);
}

function slotX(index: number, slots: number): number {
  const stepX = plotInnerWidth() / Math.max(slots - 1, 1);
  return index * stepX;
}

function latestValue(values: (number | null)[]): number {
  for (let i = values.length - 1; i >= 0; i--) {
    const value = values[i];
    if (value != null) {
      return value;
    }
  }
  return 0;
}

function pathSegments(
  values: (number | null)[],
  slots: number,
  height: number,
  max: number,
): string[] {
  const segments: string[] = [];
  let current: string[] = [];

  for (let index = 0; index < slots; index++) {
    const value = values[index];
    if (value == null) {
      if (current.length > 0) {
        segments.push(current.join(" "));
        current = [];
      }
      continue;
    }

    const x = slotX(index, slots);
    const y = valueY(value, height, max);
    current.push(`${x},${y}`);
  }

  if (current.length > 0) {
    segments.push(current.join(" "));
  }

  return segments;
}

export function LineChart({ series, height = 120, pointCount, maxValue, formatValue }: Props) {
  const slots = pointCount ?? series[0]?.values.length ?? 0;
  const numericValues = series.flatMap((s) =>
    s.values.filter((value): value is number => value != null),
  );
  const max = maxValue ?? Math.max(...numericValues, 1);
  const axisRight = plotInnerWidth();
  const baselineY = height - PAD_BOTTOM;

  return (
    <div>
      <div className="flex w-full items-stretch gap-1">
        {formatValue && (
          <div
            aria-hidden
            data-testid="chart-y-axis"
            className="flex shrink-0 flex-col justify-between py-1 text-right text-xs whitespace-nowrap text-zinc-500"
          >
            {Y_TICKS.map((fraction) => (
              <span key={fraction}>{formatValue(max * fraction)}</span>
            ))}
          </div>
        )}
        <svg
          viewBox={`0 0 ${PLOT_WIDTH} ${height}`}
          className="min-w-0 flex-1 text-zinc-500"
          role="img"
          aria-hidden
        >
          {formatValue &&
            Y_TICKS.map((fraction) => {
              const tickValue = max * fraction;
              const y = valueY(tickValue, height, max);
              return (
                <line
                  key={fraction}
                  x1={0}
                  y1={y}
                  x2={axisRight}
                  y2={y}
                  stroke="currentColor"
                  strokeOpacity={fraction === 0 ? 0.2 : 0.08}
                />
              );
            })}
          {!formatValue && (
            <line
              x1={0}
              y1={baselineY}
              x2={axisRight}
              y2={baselineY}
              stroke="currentColor"
              strokeOpacity={0.2}
            />
          )}
          {series.flatMap((s) =>
            pathSegments(s.values, slots, height, max).map((points, index) => (
              <g key={`${s.label}-${index}`} className={s.colorClass}>
                <polyline
                  fill="none"
                  stroke="currentColor"
                  strokeWidth={2}
                  strokeLinejoin="round"
                  strokeLinecap="round"
                  points={points}
                />
              </g>
            )),
          )}
        </svg>
      </div>
      <div className="mt-2 flex flex-wrap gap-4 text-xs">
        {series.map((s) => (
          <span key={s.label} className="flex items-center gap-1.5 text-zinc-400">
            <span
              className={`inline-block h-2 w-2 shrink-0 rounded-full bg-current ${s.colorClass}`}
            />
            {s.label}
            {formatValue && (
              <span className="font-medium text-zinc-200">{formatValue(latestValue(s.values))}</span>
            )}
          </span>
        ))}
      </div>
    </div>
  );
}
