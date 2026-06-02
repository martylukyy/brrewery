import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { LoginPage } from "@/pages/login";

describe("LoginPage", () => {
  it("submits credentials", async () => {
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (url === "/api/v1/auth/login") {
        return Promise.resolve({
          ok: true,
          json: async () => ({ username: "admin" }),
        });
      }
      if (url === "/api/v1/version") {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            version: "dev",
            commit: "abc",
            date: "2026-01-01",
          }),
        });
      }
      return Promise.reject(new Error(`unexpected fetch: ${url}`));
    });
    vi.stubGlobal("fetch", fetchMock);

    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={client}>
        <LoginPage />
      </QueryClientProvider>,
    );

    await userEvent.type(screen.getByLabelText(/username/i), "admin");
    await userEvent.type(screen.getByLabelText(/password/i), "password123");
    await userEvent.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/v1/auth/login",
        expect.objectContaining({ method: "POST" }),
      );
    });
  });
});
