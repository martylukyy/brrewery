import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { GaugePanel } from "@/components/gauge-panel";
import { gaugeFillClass, gaugeFillPath } from "@/lib/gauge";

describe("gaugeFillClass", () => {
  it("maps fill level to traffic-light colors", () => {
    expect(gaugeFillClass(0)).toBe("text-emerald-500");
    expect(gaugeFillClass(80)).toBe("text-emerald-500");
    expect(gaugeFillClass(80.1)).toBe("text-orange-500");
    expect(gaugeFillClass(90)).toBe("text-orange-500");
    expect(gaugeFillClass(90.1)).toBe("text-red-700");
    expect(gaugeFillClass(100)).toBe("text-red-700");
  });
});

describe("gaugeFillPath", () => {
  it("uses the short arc for partial fill so low values start on the left", () => {
    expect(gaugeFillPath(1)).toMatch(/A 42 42 0 0 1/);
    expect(gaugeFillPath(50)).toMatch(/A 42 42 0 0 1/);
  });

  it("uses the long arc for a full fill", () => {
    expect(gaugeFillPath(100)).toMatch(/A 42 42 0 1 1/);
  });
});

describe("GaugePanel", () => {
  it("renders label and display value", () => {
    render(<GaugePanel label="CPU load" value={42.5} display="42.5%" />);

    expect(screen.getByText("CPU load")).toBeInTheDocument();
    expect(screen.getByText("42.5%")).toBeInTheDocument();
  });
});
