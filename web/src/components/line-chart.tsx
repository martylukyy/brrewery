export type LineSeries = {
  label: string;
  color: string;
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

const WIDTH = 480;
const PADDING = 4;

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
  const stepX = (WIDTH - PADDING * 2) / Math.max(slots - 1, 1);
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

    const x = PADDING + index * stepX;
    const clamped = Math.min(Math.max(value, 0), max);
    const y = height - PADDING - (clamped / max) * (height - PADDING * 2);
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

  return (
    <div>
      <svg
        viewBox={`0 0 ${WIDTH} ${height}`}
        className="w-full text-zinc-500"
        role="img"
        aria-hidden
      >
        <line
          x1={PADDING}
          y1={height - PADDING}
          x2={WIDTH - PADDING}
          y2={height - PADDING}
          stroke="currentColor"
          strokeOpacity={0.2}
        />
        {series.flatMap((s) =>
          pathSegments(s.values, slots, height, max).map((points, index) => (
            <polyline
              key={`${s.label}-${index}`}
              fill="none"
              stroke={s.color}
              strokeWidth={2}
              strokeLinejoin="round"
              strokeLinecap="round"
              points={points}
            />
          )),
        )}
      </svg>
      <div className="mt-2 flex flex-wrap gap-4 text-xs">
        {series.map((s) => (
          <span key={s.label} className="flex items-center gap-1.5 text-zinc-400">
            <span className="inline-block h-2 w-2 rounded-full" style={{ background: s.color }} />
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
