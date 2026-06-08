import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { InstallSecretsModal } from "@/components/install-secrets-modal";
import type { PackageStatus } from "@/lib/api";

const packages: PackageStatus[] = [
  {
    id: "autobrr",
    name: "autobrr",
    description: "",
    category: "automation",
    install_secrets: [{
      key: "brrewery_user_password",
      label: "Brrewery password",
      type: "password",
      verify_brrewery_password: true,
    }],
    installed: false,
    dependencies_satisfied: true,
  },
];

describe("InstallSecretsModal", () => {
  it("renders a single password field for qbittorrent", () => {
    const onConfirm = vi.fn();

    render(
      <InstallSecretsModal
        packageIds={["qbittorrent"]}
        packages={[{
          id: "qbittorrent",
          name: "qBittorrent",
          description: "",
          category: "download",
          install_secrets: [{
            key: "ansible_become_password",
            label: "Password",
            type: "password",
          }],
          installed: false,
          dependencies_satisfied: true,
        }]}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.queryByLabelText("Brrewery password")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("System (sudo) password")).not.toBeInTheDocument();
  });

  it("submits entered credentials", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(
      <InstallSecretsModal
        packageIds={["autobrr"]}
        packages={packages}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.type(screen.getByLabelText("Brrewery password"), "password123");
    await user.click(screen.getByRole("button", { name: "Continue install" }));

    expect(onConfirm).toHaveBeenCalledWith({
      brrewery_user_password: "password123",
    });
  });
});
