import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { InstallOptionsModal } from "@/components/install-options-modal";
import type { PackageStatus } from "@/lib/api";

const branchVersions = ["4.4", "4.5", "4.6", "5.0", "5.1", "5.2"];

const packages: PackageStatus[] = [
  {
    id: "qbittorrent",
    name: "qBittorrent",
    description: "",
    category: "download",
    installed: false,
    dependencies_satisfied: true,
    install_options: [
      {
        key: "qbittorrent_version",
        label: "qBittorrent version",
        type: "select",
        choices: [
          { value: "4.3", label: "4.3" },
          { value: "4.6", label: "4.6" },
          { value: "5.2", label: "5.2" },
        ],
      },
      {
        key: "libtorrent_branch",
        label: "libtorrent version",
        type: "select",
        choices: [
          { value: "RC_1_2", label: "libtorrent 1.2" },
          { value: "RC_2_0", label: "libtorrent 2.0" },
        ],
        when: { key: "qbittorrent_version", one_of: branchVersions },
      },
    ],
  },
];

function renderModal(onConfirm = vi.fn()) {
  render(
    <InstallOptionsModal
      packageIds={["qbittorrent"]}
      packages={packages}
      onClose={() => {}}
      onConfirm={onConfirm}
    />,
  );
  return onConfirm;
}

describe("InstallOptionsModal", () => {
  it("skips the libtorrent branch step for the 4.3 line", async () => {
    const user = userEvent.setup();
    const onConfirm = renderModal();

    await user.click(screen.getByRole("radio", { name: "4.3" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(screen.queryByRole("radio", { name: "libtorrent 2.0" })).toBeNull();

    await user.click(screen.getByRole("button", { name: "Start install" }));
    expect(onConfirm).toHaveBeenCalledWith({ qbittorrent_version: "4.3" });
  });

  it("shows the libtorrent branch step for libtorrent-2 capable versions", async () => {
    const user = userEvent.setup();
    const onConfirm = renderModal();

    await user.click(screen.getByRole("radio", { name: "4.6" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(screen.getByRole("radio", { name: "libtorrent 1.2" })).toBeChecked();
    await user.click(screen.getByRole("radio", { name: "libtorrent 2.0" }));
    await user.click(screen.getByRole("button", { name: "Start install" }));

    expect(onConfirm).toHaveBeenCalledWith({
      qbittorrent_version: "4.6",
      libtorrent_branch: "RC_2_0",
    });
  });

  it("omits libtorrent_patch when no file is supplied", async () => {
    const user = userEvent.setup();
    const onConfirm = renderModal();

    await user.click(screen.getByRole("radio", { name: "5.2" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await user.click(screen.getByRole("button", { name: "Start install" }));

    const submitted = onConfirm.mock.calls[0][0] as Record<string, string>;
    expect(submitted).not.toHaveProperty("libtorrent_patch");
    expect(submitted.qbittorrent_version).toBe("5.2");
  });
});
