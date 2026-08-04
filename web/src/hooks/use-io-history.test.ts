import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useIOHistory } from "@/hooks/use-io-history";
import type { IOHistoryReport } from "@/lib/api";

function report(overrides: Partial<IOHistoryReport> = {}): IOHistoryReport {
  return {
    start_ms: 1_000_000,
    end_ms: 1_060_000,
    bucket_seconds: 20,
    sample_seconds: 1,
    smoothing_seconds: 5,
    series: [
      { key: "read", points: [null, 100, 200] },
      { key: "write", points: [null, 50, 60] },
    ],
    ...overrides,
  };
}

function stubFetch(body: IOHistoryReport) {
  const fetchMock = vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => body,
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return createElement(QueryClientProvider, { client }, children);
}

describe("useIOHistory", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("requests no more points than the chart is wide", async () => {
    const fetchMock = stubFetch(report());

    renderHook(() => useIOHistory("24h", 384, "/"), { wrapper });

    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    const url = String(fetchMock.mock.calls[0]?.[0]);
    expect(url).toContain("/system/io-history?");
    expect(url).toContain("range=24h");
    expect(url).toContain("points=384");
    expect(url).toContain("mount=%2F");
  });

  it("omits the mount for network throughput", async () => {
    const fetchMock = stubFetch(report({ series: [{ key: "rx", points: [1] }] }));

    renderHook(() => useIOHistory("5m", 320), { wrapper });

    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    expect(String(fetchMock.mock.calls[0]?.[0])).not.toContain("mount=");
  });

  it("indexes the returned series by key", async () => {
    stubFetch(report());

    const { result } = renderHook(() => useIOHistory("5m", 320, "/"), { wrapper });

    await waitFor(() => expect(result.current.hasSamples).toBe(true));
    expect(result.current.seriesByKey.read).toEqual([null, 100, 200]);
    expect(result.current.seriesByKey.write).toEqual([null, 50, 60]);
    // Point count follows the response, not the requested budget.
    expect(result.current.pointCount).toBe(3);
  });

  it("reports no samples while every bucket is a gap", async () => {
    stubFetch(report({ series: [{ key: "read", points: [null, null] }] }));

    const { result } = renderHook(() => useIOHistory("5m", 320, "/"), { wrapper });

    await waitFor(() => expect(result.current.seriesByKey.read).toHaveLength(2));
    expect(result.current.hasSamples).toBe(false);
  });
});
