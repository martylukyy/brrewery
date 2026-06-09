import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";

import { SESSION_QUERY_KEY } from "@/hooks/use-auth";
import { isSessionExpired } from "@/lib/api";

// createQueryClient builds the app's QueryClient with a global error handler
// that signs the user out when the session cookie expires. Any query or
// mutation that fails with a 401 (other than a deliberate credential check)
// clears the cached session, which routes the app back to the login page.
// Without this, a 401 from a background request after the cookie went stale
// would only surface as an inline error while the user stayed on a dead page.
export function createQueryClient(): QueryClient {
  const handleError = (error: unknown) => {
    if (isSessionExpired(error)) {
      queryClient.setQueryData(SESSION_QUERY_KEY, null);
    }
  };

  const queryClient = new QueryClient({
    queryCache: new QueryCache({ onError: handleError }),
    mutationCache: new MutationCache({ onError: handleError }),
  });

  return queryClient;
}
