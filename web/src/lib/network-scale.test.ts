import { describe, expect, it } from "vitest";

import { networkScaleMaxBytes } from "@/lib/network-scale";

describe("networkScaleMaxBytes", () => {
  it("converts bit/s scale to bytes/s", () => {
    expect(networkScaleMaxBytes("100mbit")).toBe(12_500_000);
    expect(networkScaleMaxBytes("1gbit")).toBe(125_000_000);
    expect(networkScaleMaxBytes("10gbit")).toBe(1_250_000_000);
  });
});
