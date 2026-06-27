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
