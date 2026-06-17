import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider, createMemoryHistory, createRouter } from "@tanstack/react-router";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { routeTree } from "@/router";

function unauthorizedFetch() {
  return vi.fn().mockResolvedValue({
    ok: false,
    status: 401,
    statusText: "Unauthorized",
    json: async () => ({ error: "Unauthorized" }),
  });
}

function renderAt(path: string, fetchImpl = unauthorizedFetch()) {
  vi.stubGlobal("fetch", fetchImpl);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [path] }),
  });
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
  return router;
}

describe("router", () => {
  it("shows the login form at /login when unauthenticated", async () => {
    renderAt("/login");

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /sign in/i })).toBeInTheDocument(),
    );
    expect(screen.queryByText(/unauthorized/i)).not.toBeInTheDocument();
  });

  it("redirects / to /login when unauthenticated", async () => {
    const router = renderAt("/");

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /sign in/i })).toBeInTheDocument(),
    );
    expect(router.state.location.pathname).toBe("/login");
  });

  it("renders the 404 page for an unknown path", async () => {
    renderAt("/notvalid");

    await waitFor(() =>
      expect(screen.getByText(/page not found/i)).toBeInTheDocument(),
    );
    expect(screen.getByRole("link", { name: /return to dashboard/i })).toBeInTheDocument();
  });
});
