import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { checkSession, login, logout, type LoginRequest } from "@/lib/api";

export const SESSION_QUERY_KEY = ["session"] as const;

export function useAuth() {
  const queryClient = useQueryClient();

  const session = useQuery({
    queryKey: SESSION_QUERY_KEY,
    queryFn: checkSession,
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
    },
  });

  return {
    isAuthenticated: session.data != null,
    isLoading: session.isPending,
    session: session.data ?? undefined,
    login: loginMutation,
    logout: logoutMutation,
  };
}
