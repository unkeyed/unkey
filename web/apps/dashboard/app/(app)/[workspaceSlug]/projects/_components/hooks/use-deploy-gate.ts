import { useFlag } from "@/lib/flags/provider";
import { trpc } from "@/lib/trpc/client";

/**
 * Whether the Unkey Deploy paywall should gate the projects screen. True only
 * when the deployBilling flag is on and the workspace has no Deploy entitlement
 * (no synced plan and no override). The authoritative gate lives in ctrl-api;
 * this drives dashboard UX only. Returns gated=false while the entitlement query
 * loads so the paywall never flashes before we know the answer.
 */
export function useDeployGate(): { gated: boolean } {
  const deployBillingEnabled = useFlag("deployBilling");
  const { data } = trpc.stripe.getDeployEntitlement.useQuery(undefined, {
    staleTime: 30_000,
  });
  return { gated: deployBillingEnabled && data !== undefined && !data.entitled };
}
