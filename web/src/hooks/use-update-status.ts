import { useQuery } from "@tanstack/react-query";

import { getUpdateStatus } from "@/lib/api";

export const UPDATE_STATUS_QUERY_KEY = ["update-status"] as const;

// useUpdateStatus surfaces the backend's cached release check for the sidebar
// badge. The backend polls GitHub on its own (6h ticker); this interval only
// picks up new ticker results, so it can be lazy.
export function useUpdateStatus() {
  return useQuery({
    queryKey: UPDATE_STATUS_QUERY_KEY,
    queryFn: () => getUpdateStatus(),
    refetchInterval: 30 * 60_000,
    retry: false,
  });
}
