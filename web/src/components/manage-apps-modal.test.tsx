import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ManageAppsModal } from "@/components/manage-apps-modal";
import type { AppStatus } from "@/lib/api";

const apps: AppStatus[] = [
  {
    id: "sonarr",
    name: "Sonarr",
    description: "TV series management",
    category: "arr",
    installed: true,
    dependencies_satisfied: true,
  },
  {
    id: "radarr",
    name: "Radarr",
    description: "Movie management",
    category: "arr",
    installed: false,
    dependencies_satisfied: true,
  },
  {
    id: "rutorrent",
    name: "ruTorrent",
    description: "Web UI for rTorrent",
    category: "download",
    installed: false,
    dependencies_satisfied: false,
  },
];

describe("ManageAppsModal", () => {
  it("lists all catalog apps", () => {
    render(<ManageAppsModal apps={apps} onClose={() => {}} onConfirm={() => {}} />);

    expect(screen.getByRole("dialog", { name: "Manage server" })).toBeInTheDocument();
    expect(screen.getByText("Sonarr")).toBeInTheDocument();
    expect(screen.getByText("Radarr")).toBeInTheDocument();
    expect(screen.getByText("ruTorrent")).toBeInTheDocument();
  });

  it("starts install for a single available app", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(<ManageAppsModal apps={apps} onClose={() => {}} onConfirm={onConfirm} />);

    await user.click(screen.getByRole("button", { name: "Install Radarr" }));

    expect(onConfirm).toHaveBeenCalledWith({
      action: "install",
      appIds: ["radarr"],
    });
  });

  it("starts upgrade for a single installed app", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(<ManageAppsModal apps={apps} onClose={() => {}} onConfirm={onConfirm} />);

    await user.click(screen.getByRole("button", { name: "Upgrade Sonarr" }));

    expect(onConfirm).toHaveBeenCalledWith({
      action: "upgrade",
      appIds: ["sonarr"],
    });
  });

  it("starts remove for a single installed app", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(<ManageAppsModal apps={apps} onClose={() => {}} onConfirm={onConfirm} />);

    await user.click(screen.getByRole("button", { name: "Remove Sonarr" }));

    expect(onConfirm).toHaveBeenCalledWith({
      action: "remove",
      appIds: ["sonarr"],
    });
  });

  it("disables actions based on app state", () => {
    render(<ManageAppsModal apps={apps} onClose={() => {}} onConfirm={() => {}} />);

    // Not installed, dependencies missing -> nothing actionable.
    expect(screen.getByRole("button", { name: "Install ruTorrent" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Upgrade ruTorrent" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Remove ruTorrent" })).toBeDisabled();

    // Not installed, dependencies satisfied -> only install.
    expect(screen.getByRole("button", { name: "Install Radarr" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Upgrade Radarr" })).toBeDisabled();

    // Installed -> upgrade and remove, but not install.
    expect(screen.getByRole("button", { name: "Install Sonarr" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Upgrade Sonarr" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Remove Sonarr" })).toBeEnabled();
  });
});
