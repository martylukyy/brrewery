import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LineChart } from "@/components/line-chart";
import { formatRate } from "@/lib/format";

describe("LineChart", () => {
  it("renders one recharts line per series", () => {
    const { container } = render(
      <LineChart
        maxValue={100}
        formatValue={formatRate}
        series={[
          { label: "Read", color: "var(--color-sky-400)", values: [0, 50] },
          { label: "Write", color: "var(--color-emerald-400)", values: [10, 20] },
        ]}
      />,
    );

    expect(container.querySelectorAll(".recharts-line")).toHaveLength(2);
  });

  it("renders Y-axis ticks formatted from maxValue", () => {
    const { container } = render(
      <LineChart
        maxValue={100}
        formatValue={(value) => `${value} B/s`}
        series={[{ label: "A", color: "var(--color-sky-400)", values: [0, 50] }]}
      />,
    );

    const ticks = Array.from(
      container.querySelectorAll(".recharts-cartesian-axis-tick-value"),
    ).map((node) => node.textContent);
    expect(ticks).toContain("0 B/s");
    expect(ticks).toContain("50 B/s");
    expect(ticks).toContain("100 B/s");
  });

  it("flattens over-scale points at the top of a fixed scale", () => {
    const { container } = render(
      <LineChart
        maxValue={100}
        formatValue={formatRate}
        series={[{ label: "Down", color: "var(--color-sky-400)", values: [50, 400, 50] }]}
      />,
    );

    const line = container.querySelector(".recharts-line-curve");
    // "M x,y C c1 c2 x,y …" — every third coordinate pair is a real point,
    // the two between are bezier control handles.
    const ys = (line?.getAttribute("d") ?? "")
      .match(/-?\d+(?:\.\d+)?,-?\d+(?:\.\d+)?/g)
      ?.map((pair) => Number(pair.split(",")[1]))
      .filter((_, index) => index % 3 === 0);

    // Y grows downward: the over-scale point sits at the plot top, level with
    // where a value of exactly maxValue would land, never above it.
    expect(ys).toHaveLength(3);
    const top = Math.min(...(ys ?? []));
    expect(ys?.[1]).toBe(top);
    expect(top).toBeGreaterThanOrEqual(0);
  });

  it("reports the raw value in the legend when it exceeds the scale", () => {
    const { getByText } = render(
      <LineChart
        maxValue={100}
        formatValue={formatRate}
        series={[{ label: "Down", color: "var(--color-sky-400)", values: [50, 400] }]}
      />,
    );

    expect(getByText("400 B/s")).toBeInTheDocument();
  });

  it("shows the latest value per series in the legend", () => {
    const { getByText } = render(
      <LineChart
        pointCount={5}
        maxValue={100}
        formatValue={formatRate}
        series={[{ label: "Down", color: "var(--color-sky-400)", values: [null, null, null, 10, 20] }]}
      />,
    );

    expect(getByText("Down")).toBeInTheDocument();
    expect(getByText("20 B/s")).toBeInTheDocument();
  });
});
