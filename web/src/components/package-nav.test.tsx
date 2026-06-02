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
];

describe("PackageNav", () => {
  it("lists only installed packages", () => {
    render(
      <PackageNav
        packages={packages}
        selectedId={null}
        onSelect={() => {}}
        onInstallClick={() => {}}
      />,
    );

    expect(screen.getByRole("button", { name: "Sonarr" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Radarr" })).not.toBeInTheDocument();
  });

  it("calls onInstallClick for add button", async () => {
    const user = userEvent.setup();
    const onInstallClick = vi.fn();

    render(
      <PackageNav
        packages={packages}
        selectedId={null}
        onSelect={() => {}}
        onInstallClick={onInstallClick}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Install packages" }));
    expect(onInstallClick).toHaveBeenCalled();
  });
});
