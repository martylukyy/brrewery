import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LineChart } from "@/components/line-chart";

describe("LineChart", () => {
  it("uses fixed maxValue for Y scale", () => {
    const { container } = render(
      <LineChart
        maxValue={100}
        series={[{ label: "A", color: "#fff", values: [0, 50, 200] }]}
      />,
    );

    const polyline = container.querySelector("polyline");
    expect(polyline).toBeTruthy();
    const points = polyline?.getAttribute("points") ?? "";
    const ys = points.split(" ").map((p) => Number(p.split(",")[1]));
    expect(ys[2]).toBeCloseTo(4, 0);
  });

  it("places the latest sample at the right edge when pointCount is fixed", () => {
    const { container } = render(
      <LineChart
        pointCount={5}
        maxValue={100}
        series={[{ label: "A", color: "#fff", values: [null, null, null, 10, 20] }]}
      />,
    );

    const polyline = container.querySelector("polyline");
    const points = polyline?.getAttribute("points") ?? "";
    const xs = points.split(" ").map((p) => Number(p.split(",")[0]));
    expect(xs[0]).toBeCloseTo(358, 0);
    expect(xs[1]).toBeCloseTo(476, 0);
  });
});
