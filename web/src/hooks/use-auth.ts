import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { checkSession, getCurrentUser, login, logout, type LoginRequest } from "@/lib/api";

export const SESSION_QUERY_KEY = ["session"] as const;

export function useAuth() {
  const queryClient = useQueryClient();

  const session = useQuery({
    queryKey: SESSION_QUERY_KEY,
    queryFn: checkSession,
    retry: false,
  });

  // The signed-in user's identity. Gated on an authenticated session so it is
  // not fetched (and does not 401) on the login screen.
  const me = useQuery({
    queryKey: ["auth", "me"],
    queryFn: getCurrentUser,
    enabled: session.data != null,
    retry: false,
  });

  const loginMutation = useMutation({
    mutationFn: (body: LoginRequest) => login(body),
    onSuccess: async () => {
      await queryClient.fetchQuery({ queryKey: SESSION_QUERY_KEY });
    },
  });

  const logoutMutation = useMutation({
    mutationFn: logout,
    onSuccess: () => {
      queryClient.setQueryData(SESSION_QUERY_KEY, null);
      queryClient.removeQueries({ queryKey: ["auth", "me"] });
    },
  });

  return {
    isAuthenticated: session.data != null,
    isLoading: session.isPending,
    session: session.data ?? undefined,
    username: me.data?.username,
    login: loginMutation,
    logout: logoutMutation,
  };
}
