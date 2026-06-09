import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useKeepaliveInterval } from "@/hooks/use-keepalive-interval";

// jsdom does not implement Worker, so these exercise the setInterval fallback.
describe("useKeepaliveInterval", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("invokes the callback once per interval", () => {
    const callback = vi.fn();
    renderHook(() => useKeepaliveInterval(callback, 1000));

    vi.advanceTimersByTime(3000);
    expect(callback).toHaveBeenCalledTimes(3);
  });

  it("always calls the latest callback without resetting the timer", () => {
    const first = vi.fn();
    const second = vi.fn();
    const { rerender } = renderHook(({ cb }) => useKeepaliveInterval(cb, 1000), {
      initialProps: { cb: first },
    });

    vi.advanceTimersByTime(1000);
    rerender({ cb: second });
    vi.advanceTimersByTime(1000);

    expect(first).toHaveBeenCalledTimes(1);
    expect(second).toHaveBeenCalledTimes(1);
  });

  it("stops ticking after unmount", () => {
    const callback = vi.fn();
    const { unmount } = renderHook(() => useKeepaliveInterval(callback, 1000));

    vi.advanceTimersByTime(1000);
    unmount();
    vi.advanceTimersByTime(3000);

    expect(callback).toHaveBeenCalledTimes(1);
  });
});
