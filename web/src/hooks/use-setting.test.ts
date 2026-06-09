import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { useSetting } from "@/hooks/use-setting";
import { SETTINGS_STORAGE_KEY } from "@/lib/settings";

function storedSettings(): Record<string, unknown> {
  const raw = window.localStorage.getItem(SETTINGS_STORAGE_KEY);
  return raw === null ? {} : JSON.parse(raw);
}

describe("useSetting", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("returns the default when nothing is stored", () => {
    const { result } = renderHook(() => useSetting("range", "fallback"));
    expect(result.current[0]).toBe("fallback");
  });

  it("hydrates the initial value from the shared settings object", () => {
    window.localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify({ range: "stored" }));
    const { result } = renderHook(() => useSetting("range", "fallback"));
    expect(result.current[0]).toBe("stored");
  });

  it("persists updates into the single brrewery_settings key", () => {
    const { result } = renderHook(() => useSetting("range", "a"));
    act(() => result.current[1]("b"));
    expect(result.current[0]).toBe("b");
    expect(storedSettings()).toEqual({ range: "b" });
  });

  it("merges fields instead of clobbering other settings", () => {
    const scale = renderHook(() => useSetting("scale", "1gbit"));
    const range = renderHook(() => useSetting("range", "days"));

    act(() => scale.result.current[1]("10gbit"));
    act(() => range.result.current[1]("months"));

    expect(storedSettings()).toEqual({ scale: "10gbit", range: "months" });
  });

  it("falls back to the default when a stored value fails validation", () => {
    window.localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify({ range: "c" }));
    const isAB = (value: unknown): value is "a" | "b" => value === "a" || value === "b";
    const { result } = renderHook(() => useSetting<"a" | "b">("range", "a", isAB));
    expect(result.current[0]).toBe("a");
  });

  it("falls back to the default when stored JSON is corrupt", () => {
    window.localStorage.setItem(SETTINGS_STORAGE_KEY, "{not valid json");
    const { result } = renderHook(() => useSetting("range", "fallback"));
    expect(result.current[0]).toBe("fallback");
  });
});
