import type { SubscriptionProduct } from "./billingSubscriptions";

/**
 * Whether a workspace keeps team access after one product's subscription is
 * deleted. Team follows any live paid product, so a delete only strips it when
 * nothing paid remains: an ending API subscription keeps team while a Deploy
 * plan is active, and an ending Deploy subscription keeps team while the API
 * tier is paid.
 *
 * The product is read straight from the deleted subscription's
 * billing_subscriptions row now, so the webhook no longer inspects columns or
 * subscription items to decide it.
 */
export function keepsTeamAfterDelete(
  product: SubscriptionProduct,
  billing: { tier: string | null; plan: string | null },
): boolean {
  return product === "api" ? billing.plan !== null : (billing.tier ?? "Free") !== "Free";
}
