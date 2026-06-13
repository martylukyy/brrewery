import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SESSION_QUERY_KEY } from "@/hooks/use-auth";
import { ApiError } from "@/lib/api";
import { createQueryClient } from "@/lib/query-client";

const session = { version: "dev", commit: "abc", date: "2026-01-01" };

describe("createQueryClient", () => {
  it("clears the cached session when a query 401s on an expired cookie", async () => {
    const client = createQueryClient();
    client.setQueryData(SESSION_QUERY_KEY, session);

    await client
      .fetchQuery({
        queryKey: ["apps"],
        queryFn: () => Promise.reject(new ApiError("Unauthorized", 401, "/apps")),
        retry: false,
      })
      .catch(() => {});

    await waitFor(() => {
      expect(client.getQueryData(SESSION_QUERY_KEY)).toBeNull();
    });
  });

  it("keeps the session when a credential check 401s (wrong password)", async () => {
    const client = createQueryClient();
    client.setQueryData(SESSION_QUERY_KEY, session);

    await client
      .getMutationCache()
      .build(client, {
        mutationFn: () =>
          Promise.reject(new ApiError("Invalid credentials", 401, "/auth/login")),
      })
      .execute(undefined)
      .catch(() => {});

    expect(client.getQueryData(SESSION_QUERY_KEY)).toEqual(session);
  });
});
