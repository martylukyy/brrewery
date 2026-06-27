import type { ReactNode } from "react";
import { PolarAngleAxis, RadialBar, RadialBarChart } from "recharts";

import { Card } from "@/components/ui/card";
import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { GAUGE_SLOT_HEIGHT_PX, gaugeColor } from "@/lib/gauge";

// 300° arc, symmetric about the vertical axis: fills from the bottom-left
// endpoint up and over the top toward the bottom-right as the value grows.
const GAUGE_START_ANGLE = 240;
const GAUGE_END_ANGLE = -60;

const chartConfig = { value: { label: "Value" } } satisfies ChartConfig;

// Below ~2.2% the fill arc is too short for two rounded corners and recharts
// draws it with square ends. Lift small non-zero readings to this minimum so
// the cap stays rounded inside the band; an exact zero still renders empty.
const MIN_VISIBLE_PERCENT = 3;

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
  const filled = clamped > 0 ? Math.max(clamped, MIN_VISIBLE_PERCENT) : 0;
  const data = [{ name: label, value: filled, fill: gaugeColor(clamped) }];

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
          <ChartContainer
            config={chartConfig}
            className="pointer-events-none aspect-auto h-full w-full"
          >
            <RadialBarChart
              data={data}
              startAngle={GAUGE_START_ANGLE}
              endAngle={GAUGE_END_ANGLE}
              innerRadius="86%"
              outerRadius="100%"
              // Decorative gauge: no focus ring or click/keyboard interaction.
              accessibilityLayer={false}
            >
              <PolarAngleAxis type="number" domain={[0, 100]} tick={false} axisLine={false} />
              <RadialBar
                dataKey="value"
                background={{ fill: "var(--color-border)" }}
                // Clamped by recharts to half the band thickness → pill caps.
                cornerRadius={9999}
                isAnimationActive={false}
              />
            </RadialBarChart>
          </ChartContainer>
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
            <span className="text-2xl font-semibold tabular-nums text-foreground">{display}</span>
          </div>
        </div>
      </div>
      <div className="flex min-h-11 w-full flex-1 flex-col items-center justify-center pt-3 text-center px-4 sm:px-8 xl:px-12">
        {footer != null && <div className="flex h-full w-full flex-1 flex-col">{footer}</div>}
      </div>
    </Card>
  );
}
