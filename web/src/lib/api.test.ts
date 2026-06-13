import { describe, expect, it } from "vitest";

import { ApiError, isSessionExpired } from "@/lib/api";

describe("isSessionExpired", () => {
  it("flags a 401 from a normal request as an expired session", () => {
    expect(isSessionExpired(new ApiError("Unauthorized", 401, "/apps"))).toBe(true);
    expect(isSessionExpired(new ApiError("Unauthorized", 401, "/version"))).toBe(true);
  });

  it("ignores a 401 from a credential check", () => {
    expect(isSessionExpired(new ApiError("Invalid credentials", 401, "/auth/login"))).toBe(false);
    expect(
      isSessionExpired(new ApiError("Incorrect password", 401, "/auth/verify-password")),
    ).toBe(false);
  });

  it("ignores non-401 errors and non-ApiError values", () => {
    expect(isSessionExpired(new ApiError("Server error", 500, "/apps"))).toBe(false);
    expect(isSessionExpired(new Error("network down"))).toBe(false);
    expect(isSessionExpired(null)).toBe(false);
  });
});
