import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InstallSecretsModal } from "@/components/install-secrets-modal";
import { ApiError, verifyPassword, type PackageStatus } from "@/lib/api";

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return { ...actual, verifyPassword: vi.fn() };
});

const mockVerify = vi.mocked(verifyPassword);

const packages: PackageStatus[] = [
  {
    id: "autobrr",
    name: "autobrr",
    description: "",
    category: "automation",
    install_secrets: [{
      key: "ansible_become_password",
      label: "Password",
      type: "password",
      verify_brrewery_password: true,
    }],
    installed: false,
    dependencies_satisfied: true,
  },
];

describe("InstallSecretsModal", () => {
  beforeEach(() => {
    mockVerify.mockReset();
    mockVerify.mockResolvedValue(undefined);
  });

  it("renders a single account password field", () => {
    render(
      <InstallSecretsModal
        packageIds={["autobrr"]}
        packages={packages}
        onClose={() => {}}
        onConfirm={vi.fn()}
      />,
    );

    expect(screen.getByLabelText("Password")).toBeInTheDocument();
  });

  it("verifies the password before submitting the entered credentials", async () => {
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

    await user.type(screen.getByLabelText("Password"), "password123");
    await user.click(screen.getByRole("button", { name: "Continue install" }));

    expect(mockVerify).toHaveBeenCalledWith("password123");
    await waitFor(() =>
      expect(onConfirm).toHaveBeenCalledWith({ ansible_become_password: "password123" }),
    );
  });

  it("shows an inline error and blocks submit when the password is wrong", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    mockVerify.mockRejectedValueOnce(new ApiError("Invalid credentials", 401));

    render(
      <InstallSecretsModal
        packageIds={["autobrr"]}
        packages={packages}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.type(screen.getByLabelText("Password"), "wrong-password");
    await user.click(screen.getByRole("button", { name: "Continue install" }));

    expect(mockVerify).toHaveBeenCalledWith("wrong-password");
    expect(await screen.findByText(/Incorrect password/i)).toBeInTheDocument();
    expect(onConfirm).not.toHaveBeenCalled();
  });
});
