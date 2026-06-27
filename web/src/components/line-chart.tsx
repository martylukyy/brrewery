import { CartesianGrid, Line, LineChart as RechartsLineChart, XAxis, YAxis } from "recharts";

import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";

export type LineSeries = {
  label: string;
  /** CSS color for the line, legend dot, and tooltip swatch (e.g. var(--color-sky-400)). */
  color: string;
  values: (number | null)[];
};

type Props = {
  series: LineSeries[];
  /** Minimum plot height; the chart grows to fill its container past this. */
  height?: number;
  /** Fixed number of X-axis slots (e.g. time-range sample count). */
  pointCount?: number;
  /** Fixed Y-axis maximum in the same units as series values. When unset, scales to data. */
  maxValue?: number;
  formatValue?: (value: number) => string;
};

function latestValue(values: (number | null)[]): number {
  for (let i = values.length - 1; i >= 0; i--) {
    const value = values[i];
    if (value != null) {
      return value;
    }
  }
  return 0;
}

export function LineChart({ series, height = 180, pointCount, maxValue, formatValue }: Props) {
  const slots = pointCount ?? series[0]?.values.length ?? 0;
  const numericValues = series.flatMap((s) =>
    s.values.filter((value): value is number => value != null),
  );
  const max = maxValue ?? Math.max(...numericValues, 1);

  // recharts keys each line by a stable dataKey; map series index -> "s0", "s1"…
  const keys = series.map((_, index) => `s${index}`);
  const config: ChartConfig = Object.fromEntries(
    series.map((s, index) => [keys[index], { label: s.label, color: s.color }]),
  );
  const data = Array.from({ length: slots }, (_, index) => {
    const row: Record<string, number | null> = { index };
    series.forEach((s, seriesIndex) => {
      row[keys[seriesIndex]] = s.values[index] ?? null;
    });
    return row;
  });

  return (
    <div className="flex h-full flex-col">
      <ChartContainer
        config={config}
        className="aspect-auto w-full flex-1"
        style={{ minHeight: height }}
      >
        <RechartsLineChart
          accessibilityLayer
          data={data}
          margin={{ top: 10, right: 8, bottom: 10, left: 0 }}
        >
          <CartesianGrid vertical={false} />
          <XAxis dataKey="index" hide />
          {formatValue ? (
            <YAxis
              domain={[0, max]}
              ticks={[0, max / 2, max]}
              interval={0}
              tickFormatter={(value) => formatValue(value)}
              tickLine={false}
              axisLine={false}
              width="auto"
            />
          ) : (
            <YAxis domain={[0, max]} hide />
          )}
          <ChartTooltip
            cursor={{ strokeOpacity: 0.2 }}
            content={
              <ChartTooltipContent
                hideLabel
                formatter={(value, name) => {
                  const item = config[name as string];
                  return (
                    <div className="flex w-full items-center gap-2">
                      <span
                        className="size-2 shrink-0 rounded-full"
                        style={{ backgroundColor: item?.color }}
                      />
                      <span className="text-muted-foreground">{item?.label ?? name}</span>
                      <span className="ml-auto font-mono font-medium tabular-nums text-foreground">
                        {value == null
                          ? "—"
                          : formatValue
                            ? formatValue(Number(value))
                            : Number(value).toLocaleString()}
                      </span>
                    </div>
                  );
                }}
              />
            }
          />
          {series.map((s, index) => (
            <Line
              key={keys[index]}
              dataKey={keys[index]}
              type="monotone"
              stroke={s.color}
              strokeWidth={2}
              dot={false}
              connectNulls={false}
              isAnimationActive={false}
            />
          ))}
        </RechartsLineChart>
      </ChartContainer>
      <div className="mt-2 flex flex-wrap gap-4 text-xs">
        {series.map((s, index) => (
          <span key={keys[index]} className="flex items-center gap-1.5 text-muted-foreground">
            <span
              className="inline-block size-2 shrink-0 rounded-full"
              style={{ backgroundColor: s.color }}
            />
            {s.label}
            {formatValue && (
              <span className="font-medium text-foreground">{formatValue(latestValue(s.values))}</span>
            )}
          </span>
        ))}
      </div>
    </div>
  );
}
