import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { checkSession, login, logout, type LoginRequest } from "@/lib/api";

export function useAuth() {
  const queryClient = useQueryClient();

  const session = useQuery({
    queryKey: ["session"],
    queryFn: checkSession,
    retry: false,
  });

  const loginMutation = useMutation({
    mutationFn: (body: LoginRequest) => login(body),
    onSuccess: async () => {
      await queryClient.fetchQuery({ queryKey: ["session"] });
    },
  });

  const logoutMutation = useMutation({
    mutationFn: logout,
    onSuccess: () => {
      queryClient.setQueryData(["session"], null);
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
