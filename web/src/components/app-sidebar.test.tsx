import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";

import { AppSidebar } from "@/components/app-sidebar";
import { SidebarProvider } from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";
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
    service: { units: ["sonarr@admin.service"], active: true, enabled: true },
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

type Overrides = Partial<ComponentProps<typeof AppSidebar>>;

function renderSidebar(overrides: Overrides = {}) {
  return render(
    <TooltipProvider>
      <SidebarProvider>
        <AppSidebar
          apps={apps}
          onManageClick={() => {}}
          onLogout={() => {}}
          onToggleService={() => {}}
          {...overrides}
        />
      </SidebarProvider>
    </TooltipProvider>,
  );
}

describe("AppSidebar", () => {
  it("lists only installed apps", () => {
    renderSidebar();

    expect(screen.getByRole("link", { name: "Sonarr" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Radarr" })).not.toBeInTheDocument();
    expect(screen.getByText("rTorrent")).toBeInTheDocument();
  });

  it("sorts installed apps alphabetically by name", () => {
    renderSidebar();

    const names = screen
      .getAllByRole("listitem")
      .map((item) => item.textContent?.trim())
      .filter((name): name is string => name === "Sonarr" || name === "rTorrent");
    expect(names).toEqual(["rTorrent", "Sonarr"]);
  });

  it("opens installed apps in a new tab", () => {
    renderSidebar();

    const link = screen.getByRole("link", { name: "Sonarr" });
    expect(link).toHaveAttribute("href", `${window.location.origin}/sonarr/`);
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("calls onManageClick for manage button", async () => {
    const user = userEvent.setup();
    const onManageClick = vi.fn();

    renderSidebar({ onManageClick });

    await user.click(screen.getByRole("button", { name: "Manage server" }));
    expect(onManageClick).toHaveBeenCalled();
  });

  it("shows the signed-in user when provided", () => {
    renderSidebar({ user: "stefan.luksch" });

    expect(screen.getByText("stefan.luksch")).toBeInTheDocument();
  });

  it("calls onLogout for log out button", async () => {
    const user = userEvent.setup();
    const onLogout = vi.fn();

    renderSidebar({ onLogout });

    await user.click(screen.getByRole("button", { name: "Log out" }));
    expect(onLogout).toHaveBeenCalled();
  });

  it("shows a service switch only for apps that expose a service", () => {
    renderSidebar();

    // Sonarr has a running service; rTorrent has none in this fixture.
    expect(screen.getByRole("switch", { name: "Stop and disable Sonarr" })).toBeChecked();
    expect(screen.queryByRole("switch", { name: /rTorrent/ })).not.toBeInTheDocument();
  });

  it("requests stop & disable when toggling a running service off", async () => {
    const user = userEvent.setup();
    const onToggleService = vi.fn();

    renderSidebar({ onToggleService });

    await user.click(screen.getByRole("switch", { name: "Stop and disable Sonarr" }));
    expect(onToggleService).toHaveBeenCalledWith(
      expect.objectContaining({ id: "sonarr" }),
      false,
    );
  });

  it("replaces the switch with a spinner while a toggle is pending", () => {
    renderSidebar({ pendingServiceAppId: "sonarr" });

    expect(screen.queryByRole("switch", { name: /Sonarr/ })).not.toBeInTheDocument();
    expect(screen.getByRole("status", { name: "Updating Sonarr service" })).toBeInTheDocument();
  });

  it("reads the switch as off for a stopped service", () => {
    renderSidebar({
      apps: [
        {
          id: "deluge",
          name: "Deluge",
          description: "",
          category: "download",
          installed: true,
          dependencies_satisfied: true,
          service: { units: ["deluged.service"], active: false, enabled: false },
        },
      ],
    });

    expect(screen.getByRole("switch", { name: "Start and enable Deluge" })).not.toBeChecked();
  });
});
