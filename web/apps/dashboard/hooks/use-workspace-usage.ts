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
    queryFn: async () =>
      (
        await getUnkeyClient().workspace.getUsage(
          period === "previous" ? { month: previousUsageMonth(new Date()) } : {},
        )
      ).data,
    retry: 1,
    ...options,
  });
}

/** The UTC month before `now`, as the `YYYY-MM` that workspace.getUsage takes */
export function previousUsageMonth(now: Date): string {
  const month = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - 1, 1));
  return `${month.getUTCFullYear()}-${String(month.getUTCMonth() + 1).padStart(2, "0")}`;
}
