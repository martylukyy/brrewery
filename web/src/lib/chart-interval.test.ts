import { describe, expect, it } from "vitest";

import {
  CHART_HISTORY_MAX_POINTS,
  getChartInterval,
  padSeriesRight,
  sliceHistoryForInterval,
} from "@/lib/chart-interval";

describe("getChartInterval", () => {
  it("returns max points for interval", () => {
    expect(getChartInterval("1m").maxPoints).toBe(60);
    expect(getChartInterval("30m").maxPoints).toBe(1800);
    expect(getChartInterval("24h").maxPoints).toBe(86_400);
    expect(CHART_HISTORY_MAX_POINTS).toBe(86_400);
  });
});

describe("sliceHistoryForInterval", () => {
  it("returns the trailing window without mutating history", () => {
    const history = Array.from({ length: 100 }, (_, index) => index);
    const sliced = sliceHistoryForInterval(history, "1m");
    expect(sliced).toHaveLength(60);
    expect(sliced[0]).toBe(40);
    expect(history).toHaveLength(100);
  });
});

describe("padSeriesRight", () => {
  it("right-aligns values in a fixed-length series", () => {
    expect(padSeriesRight([1, 2, 3], 5)).toEqual([null, null, 1, 2, 3]);
  });
});
