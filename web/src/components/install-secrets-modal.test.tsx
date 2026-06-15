import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InstallSecretsModal } from "@/components/install-secrets-modal";
import { ApiError, verifyPassword, type AppStatus } from "@/lib/api";

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return { ...actual, verifyPassword: vi.fn() };
});

const mockVerify = vi.mocked(verifyPassword);

const apps: AppStatus[] = [
  {
    id: "autobrr",
    name: "autobrr",
    description: "",
    category: "automation",
    install_secrets: [
      {
        key: "ansible_become_password",
        label: "Password",
        type: "password",
        verify_brrewery_password: true,
      },
      // Synthetic app-specific secret. No catalog manifest declares one today,
      // but the modal supports them, so this exercises the install-only path
      // (app credentials are collected on install, skipped on upgrade/remove).
      {
        key: "example_app_credential",
        label: "App credential",
        type: "text",
      },
    ],
    installed: false,
    dependencies_satisfied: true,
  },
];

describe("InstallSecretsModal", () => {
  beforeEach(() => {
    mockVerify.mockReset();
    mockVerify.mockResolvedValue(undefined);
  });

  it("renders every declared install secret for install", () => {
    render(
      <InstallSecretsModal
        action="install"
        appIds={["autobrr"]}
        apps={apps}
        onClose={() => {}}
        onConfirm={vi.fn()}
      />,
    );

    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByLabelText("App credential")).toBeInTheDocument();
  });

  it("verifies the password before submitting the entered credentials", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(
      <InstallSecretsModal
        action="install"
        appIds={["autobrr"]}
        apps={apps}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.type(screen.getByLabelText("Password"), "password123");
    await user.type(screen.getByLabelText("App credential"), "cred-abc");
    await user.click(screen.getByRole("button", { name: "Continue install" }));

    expect(mockVerify).toHaveBeenCalledWith("password123");
    await waitFor(() =>
      expect(onConfirm).toHaveBeenCalledWith({
        ansible_become_password: "password123",
        example_app_credential: "cred-abc",
      }),
    );
  });

  it.each([
    ["upgrade", "Continue upgrade"] as const,
    ["remove", "Continue remove"] as const,
  ])("only asks for the account password to %s", async (action, submitLabel) => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(
      <InstallSecretsModal
        action={action}
        appIds={["autobrr"]}
        apps={apps}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.queryByLabelText("App credential")).not.toBeInTheDocument();

    await user.type(screen.getByLabelText("Password"), "password123");
    await user.click(screen.getByRole("button", { name: submitLabel }));

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
        action="install"
        appIds={["autobrr"]}
        apps={apps}
        onClose={() => {}}
        onConfirm={onConfirm}
      />,
    );

    await user.type(screen.getByLabelText("Password"), "wrong-password");
    await user.type(screen.getByLabelText("App credential"), "token-abc");
    await user.click(screen.getByRole("button", { name: "Continue install" }));

    expect(mockVerify).toHaveBeenCalledWith("wrong-password");
    expect(await screen.findByText(/Incorrect password/i)).toBeInTheDocument();
    expect(onConfirm).not.toHaveBeenCalled();
  });
});
