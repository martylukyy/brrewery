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
});
