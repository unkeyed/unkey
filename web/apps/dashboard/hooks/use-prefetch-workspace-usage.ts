import { workspaceUsageQuery } from "@/hooks/use-workspace-usage";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { trpc } from "@/lib/trpc/client";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

/**
 * Warms the queries behind the settings Usage page. Each call is a no-op while
 * the cached data is still fresh.
 */
export function usePrefetchWorkspaceUsage(): () => void {
  const billingUpgrades = useBillingUIUpgrades();
  const queryClient = useQueryClient();
  const trpcUtils = trpc.useUtils();

  return useCallback(() => {
    if (!billingUpgrades) {
      return;
    }
    void queryClient.prefetchQuery(workspaceUsageQuery("current"));
    void trpcUtils.stripe.getBillingInfo.prefetch(undefined, { staleTime: 30_000 });
  }, [billingUpgrades, queryClient, trpcUtils]);
}
