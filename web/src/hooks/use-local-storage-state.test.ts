import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { useLocalStorageState } from "@/hooks/use-local-storage-state";

describe("useLocalStorageState", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("returns the default when nothing is stored", () => {
    const { result } = renderHook(() => useLocalStorageState("key", "fallback"));
    expect(result.current[0]).toBe("fallback");
  });

  it("hydrates the initial value from localStorage", () => {
    window.localStorage.setItem("key", JSON.stringify("stored"));
    const { result } = renderHook(() => useLocalStorageState("key", "fallback"));
    expect(result.current[0]).toBe("stored");
  });

  it("persists updates back to localStorage", () => {
    const { result } = renderHook(() => useLocalStorageState("key", "a"));
    act(() => result.current[1]("b"));
    expect(result.current[0]).toBe("b");
    expect(window.localStorage.getItem("key")).toBe(JSON.stringify("b"));
  });

  it("falls back to the default when the stored value fails validation", () => {
    window.localStorage.setItem("key", JSON.stringify("c"));
    const isAB = (value: unknown): value is "a" | "b" => value === "a" || value === "b";
    const { result } = renderHook(() => useLocalStorageState<"a" | "b">("key", "a", isAB));
    expect(result.current[0]).toBe("a");
  });

  it("falls back to the default when stored JSON is corrupt", () => {
    window.localStorage.setItem("key", "{not valid json");
    const { result } = renderHook(() => useLocalStorageState("key", "fallback"));
    expect(result.current[0]).toBe("fallback");
  });
});
