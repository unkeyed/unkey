import type Stripe from "stripe";
import type { DeployBillingConfig } from "./deployBilling";

/** The subscription shape a cancel encountered, for logging and tests. */
export type DeployCancelTopology = "none" | "deploy_only" | "mixed";

/**
 * Stops the Stripe renewal for a Deploy subscription, no refund. Deploy-only
 * subscriptions cancel at period end; mixed ones (Deploy + API) keep their
 * metered items and drop only the plan-fee item(s), so usage bills at the
 * boundary then reports zero. Returns the topology; "none" is an idempotent
 * no-op. Lives here rather than in ctrl so Stripe knowledge stays in one place.
 */
export async function cancelDeploySubscription(
  stripe: Stripe,
  subscriptionId: string,
  config: DeployBillingConfig,
): Promise<DeployCancelTopology> {
  const sub = await stripe.subscriptions.retrieve(subscriptionId);
  const planFeePriceIds = new Set<string>(Object.values(config.planFeePriceIds));

  let deployItems = 0;
  const planFeeItemIds: string[] = [];
  for (const item of sub.items.data) {
    const priceId = item.price?.id ?? "";
    if (config.allDeployPriceIds.has(priceId)) {
      deployItems++;
    }
    if (planFeePriceIds.has(priceId)) {
      planFeeItemIds.push(item.id);
    }
  }

  if (deployItems === 0) {
    return "none";
  }

  if (deployItems === sub.items.data.length) {
    await stripe.subscriptions.update(subscriptionId, { cancel_at_period_end: true });
    return "deploy_only";
  }

  await stripe.subscriptions.update(subscriptionId, {
    items: planFeeItemIds.map((id) => ({ id, deleted: true })),
    proration_behavior: "none",
  });
  return "mixed";
}
