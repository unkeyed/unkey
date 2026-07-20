/**
 * Which product a Stripe subscription event belongs to, decided purely from the
 * workspace_billing column its subscription id was found in. After the billing
 * split each product owns a whole subscription, so the webhook does one OR
 * lookup on both columns and branches on the matched column here instead of
 * inspecting the subscription's items.
 */
export type MatchedSubscriptionColumn = "api" | "deploy";

/**
 * The column a subscription id matched, or null when the row points at neither.
 * The API column wins a tie: the two ids are always distinct in practice, and a
 * degenerate row with both set to the same id is still routed deterministically.
 */
export function matchSubscriptionColumn(
  billing: { stripeSubscriptionId: string | null; stripeDeploySubscriptionId: string | null },
  subscriptionId: string,
): MatchedSubscriptionColumn | null {
  if (billing.stripeSubscriptionId === subscriptionId) {
    return "api";
  }
  if (billing.stripeDeploySubscriptionId === subscriptionId) {
    return "deploy";
  }
  return null;
}

/**
 * Whether a workspace keeps team access after one product's subscription is
 * deleted. Team follows any live paid product, so a delete only strips it when
 * nothing paid remains: an ending API subscription keeps team while a Deploy
 * plan is active, and an ending Deploy subscription keeps team while the API
 * tier is paid.
 */
export function keepsTeamAfterDelete(
  column: MatchedSubscriptionColumn,
  billing: { tier: string | null; plan: string | null },
): boolean {
  return column === "api" ? billing.plan !== null : (billing.tier ?? "Free") !== "Free";
}
