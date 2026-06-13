import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AppNav } from "@/components/app-nav";
import type { AppStatus } from "@/lib/api";

const apps: AppStatus[] = [
  {
    id: "sonarr",
    name: "Sonarr",
    description: "",
    category: "arr",
    icon: "/apps/sonarr.png",
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

describe("AppNav", () => {
  it("lists only installed apps", () => {
    render(
      <AppNav
        apps={apps}
        onManageClick={() => {}}
      />,
    );

    expect(screen.getByRole("link", { name: "Sonarr" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Radarr" })).not.toBeInTheDocument();
    expect(screen.getByText("rTorrent")).toBeInTheDocument();
  });

  it("opens installed apps in a new tab", () => {
    render(
      <AppNav
        apps={apps}
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
      <AppNav
        apps={apps}
        onManageClick={onManageClick}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Manage server" }));
    expect(onManageClick).toHaveBeenCalled();
  });
});
