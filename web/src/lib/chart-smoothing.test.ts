import { describe, expect, it } from "vitest";

import { smoothSeries } from "@/lib/chart-smoothing";

describe("smoothSeries", () => {
  it("averages over the trailing five samples", () => {
    const values = [0, 0, 0, 0, 10, 0, 0, 0, 0, 0];
    expect(smoothSeries(values, 1000)).toEqual([0, 0, 0, 0, 2, 2, 2, 2, 2, 0]);
  });

  it("averages over the samples available at the leading edge", () => {
    expect(smoothSeries([4, 8], 1000)).toEqual([4, 6]);
  });

  it("keeps a flat series flat", () => {
    expect(smoothSeries([5, 5, 5, 5, 5, 5, 5], 1000)).toEqual([5, 5, 5, 5, 5, 5, 5]);
  });

  it("returns values unchanged when the window spans a single sample", () => {
    const values = [1, 9, 2, 8];
    expect(smoothSeries(values, 5000)).toEqual(values);
  });

  it("handles an empty series", () => {
    expect(smoothSeries([], 1000)).toEqual([]);
  });
});
