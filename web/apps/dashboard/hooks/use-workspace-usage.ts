import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import type { V2WorkspaceGetUsageResponseData } from "@unkey/api/models/components";

type UsagePeriod = "current" | "previous";

/**
 * The workspace usage of one month. Every caller of a period shares one cached
 * response, so each caller only picks how fresh it must be.
 */
export function useWorkspaceUsage(
  period: UsagePeriod,
  options: { enabled?: boolean; staleTime?: number; refetchOnWindowFocus?: boolean } = {},
) {
  return useQuery<V2WorkspaceGetUsageResponseData, Error>({
    queryKey: queryKeys.workspace.usage(period),
    queryFn: async () => (await getUnkeyClient().workspace.getUsage({ period })).data,
    retry: 1,
    ...options,
  });
}
