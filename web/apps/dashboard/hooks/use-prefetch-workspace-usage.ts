import { workspaceUsageQuery } from "@/hooks/use-workspace-usage";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useQueryClient } from "@tanstack/react-query";
import { getQueryKey } from "@trpc/react-query";
import { useCallback } from "react";

/**
 * Warms the queries behind the settings Usage page. Each call is a no-op while
 * the cached data is still fresh.
 */
export function usePrefetchWorkspaceUsage(): () => void {
  const billingUpgrades = useBillingUIUpgrades();
  const { user } = useWorkspace();
  const queryClient = useQueryClient();
  const trpcUtils = trpc.useUtils();

  return useCallback(() => {
    if (!billingUpgrades || !user?.orgId) {
      return;
    }
    void queryClient.prefetchQuery(workspaceUsageQuery("current", user.orgId, user.id));
    void queryClient.prefetchQuery({
      queryKey: getQueryKey(trpc.stripe.getBillingInfo, undefined, "query"),
      queryFn: () =>
        trpcUtils.client.stripe.getBillingInfo.query(undefined, {
          context: { skipBatch: true },
        }),
      staleTime: 30_000,
    });
  }, [billingUpgrades, queryClient, trpcUtils, user]);
}
