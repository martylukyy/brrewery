import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { GaugePanel } from "@/components/gauge-panel";
import { gaugeColor } from "@/lib/gauge";

describe("gaugeColor", () => {
  it("maps fill level to traffic-light colors", () => {
    expect(gaugeColor(0)).toBe("var(--gauge-ok)");
    expect(gaugeColor(80)).toBe("var(--gauge-ok)");
    expect(gaugeColor(80.1)).toBe("var(--gauge-warn)");
    expect(gaugeColor(90)).toBe("var(--gauge-warn)");
    expect(gaugeColor(90.1)).toBe("var(--gauge-crit)");
    expect(gaugeColor(100)).toBe("var(--gauge-crit)");
  });
});

describe("GaugePanel", () => {
  it("renders label and display value", () => {
    render(<GaugePanel label="CPU load" value={42.5} display="42.5%" />);

    expect(screen.getByText("CPU load")).toBeInTheDocument();
    expect(screen.getByText("42.5%")).toBeInTheDocument();
  });

  it("renders a recharts radial gauge tinted by the value", () => {
    const { container } = render(<GaugePanel label="CPU" value={95} display="95%" />);

    const sector = container.querySelector(".recharts-radial-bar-sectors path");
    expect(sector).toBeTruthy();
    expect(sector?.getAttribute("fill")).toBe("var(--gauge-crit)");
  });
});
