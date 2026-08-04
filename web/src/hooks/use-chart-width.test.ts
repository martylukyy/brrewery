import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useChartWidth } from "@/hooks/use-chart-width";

type Observed = { callback: ResizeObserverCallback; disconnected: boolean };

function stubResizeObserver(): Observed[] {
  const observers: Observed[] = [];
  vi.stubGlobal(
    "ResizeObserver",
    class {
      private entry: Observed;

      constructor(callback: ResizeObserverCallback) {
        this.entry = { callback, disconnected: false };
        observers.push(this.entry);
      }

      observe() {}
      unobserve() {}
      disconnect() {
        this.entry.disconnected = true;
      }
    },
  );
  return observers;
}

function resizeTo(observer: Observed, width: number) {
  act(() => {
    observer.callback(
      [{ contentRect: { width } } as ResizeObserverEntry],
      {} as ResizeObserver,
    );
  });
}

describe("useChartWidth", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("quantizes the measured width so resizing does not churn", () => {
    const observers = stubResizeObserver();
    const { result } = renderHook(() => useChartWidth());

    act(() => result.current.ref(document.createElement("div")));

    resizeTo(observers[0], 811);
    expect(result.current.width).toBe(800);

    // A few pixels either way keeps the same budget.
    resizeTo(observers[0], 799);
    expect(result.current.width).toBe(768);
  });

  it("reports a usable width before anything is measured", () => {
    stubResizeObserver();
    const { result } = renderHook(() => useChartWidth());

    expect(result.current.width).toBeGreaterThan(0);
  });

  it("stops observing the previous element", () => {
    const observers = stubResizeObserver();
    const { result } = renderHook(() => useChartWidth());

    act(() => result.current.ref(document.createElement("div")));
    act(() => result.current.ref(document.createElement("div")));

    expect(observers[0].disconnected).toBe(true);
    expect(observers).toHaveLength(2);
  });
});
