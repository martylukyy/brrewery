import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { PackageNav } from "@/components/package-nav";
import type { PackageStatus } from "@/lib/api";

const packages: PackageStatus[] = [
  {
    id: "sonarr",
    name: "Sonarr",
    description: "",
    category: "arr",
    icon: "/packages/sonarr.png",
    web_path: "/sonarr/",
    installed: true,
    dependencies_satisfied: true,
  },
  {
    id: "radarr",
    name: "Radarr",
    description: "",
    category: "arr",
    installed: false,
    dependencies_satisfied: true,
  },
  {
    id: "rtorrent",
    name: "rTorrent",
    description: "",
    category: "download",
    installed: true,
    dependencies_satisfied: true,
  },
];

describe("PackageNav", () => {
  it("lists only installed packages", () => {
    render(
      <PackageNav
        packages={packages}
        onManageClick={() => {}}
      />,
    );

    expect(screen.getByRole("link", { name: "Sonarr" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Radarr" })).not.toBeInTheDocument();
    expect(screen.getByText("rTorrent")).toBeInTheDocument();
  });

  it("opens installed apps in a new tab", () => {
    render(
      <PackageNav
        packages={packages}
        onManageClick={() => {}}
      />,
    );

    const link = screen.getByRole("link", { name: "Sonarr" });
    expect(link).toHaveAttribute("href", `${window.location.origin}/sonarr/`);
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("calls onManageClick for manage button", async () => {
    const user = userEvent.setup();
    const onManageClick = vi.fn();

    render(
      <PackageNav
        packages={packages}
        onManageClick={onManageClick}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Manage packages" }));
    expect(onManageClick).toHaveBeenCalled();
  });
});
