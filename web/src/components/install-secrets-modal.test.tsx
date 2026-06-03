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
