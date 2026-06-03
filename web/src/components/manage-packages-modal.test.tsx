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

  it("starts install for selected available packages", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(
      <ManagePackagesModal
        packages={packages}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByRole("checkbox", { name: /Radarr/i }));
    await user.click(screen.getByRole("button", { name: "Install" }));

    expect(onConfirm).toHaveBeenCalledWith({
      action: "install",
      packageIds: ["radarr"],
    });
  });

  it("starts upgrade for selected installed packages", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(
      <ManagePackagesModal
        packages={packages}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByRole("checkbox", { name: /Sonarr/i }));
    await user.click(screen.getByRole("button", { name: "Upgrade" }));

    expect(onConfirm).toHaveBeenCalledWith({
      action: "upgrade",
      packageIds: ["sonarr"],
    });
  });

  it("starts remove for selected installed packages", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(
      <ManagePackagesModal
        packages={packages}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByRole("checkbox", { name: /Sonarr/i }));
    await user.click(screen.getByRole("button", { name: "Remove" }));

    expect(onConfirm).toHaveBeenCalledWith({
      action: "remove",
      packageIds: ["sonarr"],
    });
  });

  it("disables install when dependencies are not satisfied", async () => {
    const user = userEvent.setup();

    render(
      <ManagePackagesModal
        packages={packages}
        onClose={() => {}}
        onConfirm={() => {}}
      />,
    );

    await user.click(screen.getByRole("checkbox", { name: /ruTorrent/i }));
    expect(screen.getByRole("button", { name: "Install" })).toBeDisabled();
  });
});
