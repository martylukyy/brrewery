import { describe, expect, it } from "vitest";

import { appUrl } from "@/lib/app-link";

describe("appUrl", () => {
  it("returns null when no web path is set", () => {
    expect(appUrl(undefined)).toBeNull();
    expect(appUrl("")).toBeNull();
  });

  it("resolves a same-origin reverse-proxy path against the current origin", () => {
    expect(appUrl("/sonarr/")).toBe(`${window.location.origin}/sonarr/`);
  });

  it("links a port-form path to the current host on that port and scheme", () => {
    expect(appUrl("http://:32400/web")).toBe(
      `http://${window.location.hostname}:32400/web`,
    );
  });
});
