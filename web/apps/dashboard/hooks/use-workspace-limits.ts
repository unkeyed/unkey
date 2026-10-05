import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import type { V2WorkspaceGetLimitsResponseData } from "@unkey/api/models/components";

/**
 * The workspace limits with current usage. Every caller shares one cached
 * response, so each caller only picks how fresh it must be.
 */
export function useWorkspaceLimits(
  options: { enabled?: boolean; staleTime?: number; refetchOnWindowFocus?: boolean } = {},
) {
  return useQuery<V2WorkspaceGetLimitsResponseData, Error>({
    queryKey: queryKeys.workspace.limits,
    queryFn: async () => (await getUnkeyClient().workspace.getLimits()).data,
    retry: 1,
    ...options,
  });
}
