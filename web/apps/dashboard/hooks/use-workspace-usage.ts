import {
  type UsagePeriod,
  usagePeriodMonth,
} from "@/app/(app)/[workspaceSlug]/settings/usage/period";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import type { V2WorkspaceGetUsageResponseData } from "@unkey/api/models/components";

export function workspaceUsageQuery(period: UsagePeriod) {
  return {
    queryKey: queryKeys.workspace.usage(period),
    queryFn: async (): Promise<V2WorkspaceGetUsageResponseData> =>
      (
        await getUnkeyClient().workspace.getUsage(
          period === "previous" ? { period: usagePeriodMonth(period, new Date()) } : {},
        )
      ).data,
    retry: 1,
  };
}

/**
 * The workspace usage of one month. Every caller of a period shares one cached
 * response, so each caller only picks how fresh it must be.
 */
export function useWorkspaceUsage(
  period: UsagePeriod,
  options: { enabled?: boolean; staleTime?: number; refetchOnWindowFocus?: boolean } = {},
) {
  return useQuery<V2WorkspaceGetUsageResponseData, Error>({
    ...workspaceUsageQuery(period),
    ...options,
  });
}
