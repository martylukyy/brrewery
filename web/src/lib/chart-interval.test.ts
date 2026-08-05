import { describe, expect, it } from "vitest";

import {
  chartPointBudget,
  getChartInterval,
  isChartIntervalId,
} from "@/lib/chart-interval";

describe("getChartInterval", () => {
  it("returns the range length in seconds", () => {
    expect(getChartInterval("1m").seconds).toBe(60);
    expect(getChartInterval("30m").seconds).toBe(1800);
    expect(getChartInterval("24h").seconds).toBe(86_400);
  });
});

describe("isChartIntervalId", () => {
  it("accepts known ids only", () => {
    expect(isChartIntervalId("24h")).toBe(true);
    expect(isChartIntervalId("7d")).toBe(false);
  });
});

describe("chartPointBudget", () => {
  it("never asks for more points than the chart is wide", () => {
    expect(chartPointBudget("24h", 640)).toBe(640);
    expect(chartPointBudget("24h", 1920)).toBe(1920);
  });

  it("never asks for a finer resolution than the sample rate", () => {
    // A 1 minute range holds 60 one-second samples, however wide the plot is.
    expect(chartPointBudget("1m", 640)).toBe(60);
  });

  it("stays within the daemon's per-request cap", () => {
    expect(chartPointBudget("24h", 10_000)).toBe(4000);
  });

  it("returns at least one point for a zero-width plot", () => {
    expect(chartPointBudget("5m", 0)).toBe(1);
  });
});
