import { useFlag } from "@/lib/flags/provider";
import { trpc } from "@/lib/trpc/client";

/**
 * Gates the projects screen behind the Compute paywall (deployBilling on + no
 * Deploy entitlement). ctrl-api is the real gate; this is UX only and fails
 * open: a loading or errored query leaves `gated` false rather than blocking.
 */
export function useDeployGate(): { gated: boolean; isLoading: boolean } {
  const deployBillingEnabled = useFlag("deployBilling");
  const { data, isLoading } = trpc.stripe.getDeployEntitlement.useQuery(undefined, {
    staleTime: 30_000,
  });
  return {
    gated: deployBillingEnabled && data !== undefined && !data.entitled,
    isLoading: deployBillingEnabled && isLoading,
  };
}
