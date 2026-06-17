import { render, screen, within } from "@testing-library/react";
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
    service: { units: ["sonarr@admin.service"], active: true, enabled: true, failing: false },
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
          service: { units: ["deluged.service"], active: false, enabled: false, failing: false },
        },
      ],
    });

    expect(screen.getByRole("switch", { name: "Start and enable Deluge" })).not.toBeChecked();
  });

  it("does not flag a healthy running service as failing", () => {
    renderSidebar();

    // Sonarr is active+enabled: the accessible name has no failing suffix, the
    // switch is not forced red, and the thumb carries no "!".
    const sonarr = screen.getByRole("switch", { name: "Stop and disable Sonarr" });
    expect(sonarr).not.toHaveClass("bg-red-600!");
    expect(screen.queryByText("!")).not.toBeInTheDocument();
  });

  it("turns the switch's own track red and names the failing state when a unit is failing", () => {
    renderSidebar({
      apps: [
        {
          id: "deluge",
          name: "Deluge",
          description: "",
          category: "download",
          installed: true,
          dependencies_satisfied: true,
          // deluge-web crash-looping: enabled but never reaches running, so the
          // switch reads off while the red track flags the unhealthy service.
          service: {
            units: ["deluged@admin.service", "deluge-web@admin.service"],
            active: false,
            enabled: true,
            failing: true,
          },
        },
      ],
    });

    const sw = screen.getByRole("switch", { name: "Start and enable Deluge (service failing)" });
    expect(sw).toHaveClass("bg-red-600!");
    // The thumb carries a "!" to mark the unhealthy service.
    expect(within(sw).getByText("!")).toBeInTheDocument();
  });

  // rTorrent (service, no web UI) and ruTorrent (web UI, no service) are two
  // halves of one thing, so the sidebar collapses them into a single row.
  describe("rTorrent + ruTorrent", () => {
    const combinedApps: AppStatus[] = [
      {
        id: "rtorrent",
        name: "rTorrent",
        description: "",
        category: "download",
        icon: "/apps/rutorrent.png",
        installed: true,
        dependencies_satisfied: true,
        service: { units: ["rtorrent@admin.service"], active: true, enabled: true, failing: false },
      },
      {
        id: "rutorrent",
        name: "ruTorrent",
        description: "",
        category: "download",
        icon: "/apps/rutorrent.png",
        web_path: "/rutorrent/",
        installed: true,
        dependencies_satisfied: true,
      },
    ];

    it("collapses the pair into a single r(u)Torrent row linking to ruTorrent", () => {
      renderSidebar({ apps: combinedApps });

      const link = screen.getByRole("link", { name: "r(u)Torrent" });
      expect(link).toHaveAttribute("href", `${window.location.origin}/rutorrent/`);
      // Neither half is listed on its own.
      expect(screen.queryByText("rTorrent")).not.toBeInTheDocument();
      expect(screen.queryByText("ruTorrent")).not.toBeInTheDocument();
    });

    it("toggles rTorrent's service from the combined row", async () => {
      const user = userEvent.setup();
      const onToggleService = vi.fn();

      renderSidebar({ apps: combinedApps, onToggleService });

      // The switch reads rTorrent's running service but is labelled with the
      // combined name; the toggle still targets rTorrent's id underneath.
      await user.click(screen.getByRole("switch", { name: "Stop and disable r(u)Torrent" }));
      expect(onToggleService).toHaveBeenCalledWith(
        expect.objectContaining({ id: "rtorrent" }),
        false,
      );
    });

    it("spins the combined switch while rTorrent's toggle is pending", () => {
      renderSidebar({ apps: combinedApps, pendingServiceAppId: "rtorrent" });

      expect(screen.queryByRole("switch")).not.toBeInTheDocument();
      expect(
        screen.getByRole("status", { name: "Updating r(u)Torrent service" }),
      ).toBeInTheDocument();
    });

    it("shows rTorrent on its own when ruTorrent is not installed", () => {
      renderSidebar({ apps: [combinedApps[0]] });

      // Falls back to its own identity: no web UI to link to, but the service
      // switch is still present.
      expect(screen.getByText("rTorrent")).toBeInTheDocument();
      expect(screen.queryByRole("link")).not.toBeInTheDocument();
      expect(
        screen.getByRole("switch", { name: "Stop and disable rTorrent" }),
      ).toBeInTheDocument();
    });
  });
});
