import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ManagePackagesModal } from "@/components/manage-packages-modal";
import type { PackageStatus } from "@/lib/api";

const packages: PackageStatus[] = [
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

describe("ManagePackagesModal", () => {
  it("lists all catalog packages", () => {
    render(
      <ManagePackagesModal
        packages={packages}
        onClose={() => {}}
        onConfirm={() => {}}
      />,
    );

    expect(screen.getByRole("dialog", { name: "Manage packages" })).toBeInTheDocument();
    expect(screen.getByText("Sonarr")).toBeInTheDocument();
    expect(screen.getByText("Radarr")).toBeInTheDocument();
    expect(screen.getByText("ruTorrent")).toBeInTheDocument();
  });

  it("starts install for a single available package", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(
      <ManagePackagesModal
        packages={packages}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Install Radarr" }));

    expect(onConfirm).toHaveBeenCalledWith({
      action: "install",
      packageIds: ["radarr"],
    });
  });

  it("starts upgrade for a single installed package", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(
      <ManagePackagesModal
        packages={packages}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Upgrade Sonarr" }));

    expect(onConfirm).toHaveBeenCalledWith({
      action: "upgrade",
      packageIds: ["sonarr"],
    });
  });

  it("starts remove for a single installed package", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(
      <ManagePackagesModal
        packages={packages}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Remove Sonarr" }));

    expect(onConfirm).toHaveBeenCalledWith({
      action: "remove",
      packageIds: ["sonarr"],
    });
  });

  it("disables actions based on package state", () => {
    render(
      <ManagePackagesModal
        packages={packages}
        onClose={() => {}}
        onConfirm={() => {}}
      />,
    );

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
