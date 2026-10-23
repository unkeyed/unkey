import {
  type UsagePeriod,
  usagePeriodMonth,
} from "@/app/(app)/[workspaceSlug]/settings/usage/period";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useQuery } from "@tanstack/react-query";
import type { V2WorkspaceGetUsageResponseData } from "@unkey/api/models/components";

export function workspaceUsageQuery(
  period: UsagePeriod,
  orgId: string | null,
  userId: string | null,
) {
  return {
    queryKey: [...queryKeys.workspace.usage(period), orgId, userId],
    queryFn: async ({
      signal,
    }: { signal?: AbortSignal }): Promise<V2WorkspaceGetUsageResponseData> =>
      (
        await getUnkeyClient().workspace.getUsage(
          period === "previous" ? { period: usagePeriodMonth(period, new Date()) } : {},
          { signal },
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
  const { user } = useWorkspace();
  return useQuery<V2WorkspaceGetUsageResponseData, Error>({
    ...workspaceUsageQuery(period, user?.orgId ?? null, user?.id ?? null),
    ...options,
    enabled: Boolean(user?.orgId && user.id) && options.enabled !== false,
    keepPreviousData: false,
  });
}
