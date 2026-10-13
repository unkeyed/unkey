import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useQuery } from "@tanstack/react-query";
import type { V2WorkspaceGetLimitsResponseData } from "@unkey/api/models/components";

/**
 * The workspace limits with current usage. Every caller shares one cached
 * response, so each caller only picks how fresh it must be.
 */
export function useWorkspaceLimits(
  options: {
    enabled?: boolean;
    staleTime?: number;
    refetchInterval?: number;
    refetchOnWindowFocus?: boolean;
  } = {},
) {
  const { user } = useWorkspace();
  return useQuery<V2WorkspaceGetLimitsResponseData, Error>({
    queryKey: [...queryKeys.workspace.limits, user?.orgId, user?.id],
    queryFn: async ({ signal }) => (await getUnkeyClient().workspace.getLimits({ signal })).data,
    retry: 1,
    ...options,
    enabled: Boolean(user?.orgId && user.id) && options.enabled !== false,
    keepPreviousData: false,
  });
}
