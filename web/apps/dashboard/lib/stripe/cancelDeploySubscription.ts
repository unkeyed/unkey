import type Stripe from "stripe";
import type { DeployBillingConfig } from "./deployBilling";
import { getApiCancelSchedule, isPendingSchedule } from "./subscriptionUtils";

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

  // Mixed subscription. A pending API-cancel schedule (created by
  // cancelSubscription) already ends the API plan at period end, and its phase 2
  // snapshots the CURRENT items including the Compute plan fee. Editing items
  // directly here would either be rejected (Stripe forbids item-level edits on a
  // scheduled subscription) or silently undone when phase 2 reinstates the plan
  // fee at the boundary, re-billing Compute and re-entitling a workspace whose
  // compute was already torn down. Since the API plan is already cancelling and
  // the user is now cancelling Compute, both products are gone: release the
  // schedule and cancel the whole subscription at period end. Metered usage bills
  // to the boundary then stops; neither plan fee renews.
  const apiCancelSchedule = await getApiCancelSchedule(stripe, sub);
  if (apiCancelSchedule && isPendingSchedule(apiCancelSchedule)) {
    await stripe.subscriptionSchedules.release(apiCancelSchedule.id);
    await stripe.subscriptions.update(subscriptionId, { cancel_at_period_end: true });
    return "mixed";
  }

  await stripe.subscriptions.update(subscriptionId, {
    items: planFeeItemIds.map((id) => ({ id, deleted: true })),
    proration_behavior: "none",
  });
  return "mixed";
}
