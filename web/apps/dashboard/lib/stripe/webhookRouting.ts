import type { SubscriptionProduct } from "./billingSubscriptions";
import { deployPlanGrantsTeam } from "./deployPlan";

/**
 * Whether a workspace keeps team access after one product's subscription is
 * deleted. Paid API tiers and the Compute Pro/Business plans grant team access;
 * Compute Starter does not.
 *
 * The product is read straight from the deleted subscription's
 * billing_subscriptions row now, so the webhook no longer inspects columns or
 * subscription items to decide it.
 */
export function keepsTeamAfterDelete(
  product: SubscriptionProduct,
  billing: { tier: string | null; plan: string | null },
): boolean {
  return product === "api"
    ? deployPlanGrantsTeam(billing.plan)
    : (billing.tier ?? "Free") !== "Free";
}
