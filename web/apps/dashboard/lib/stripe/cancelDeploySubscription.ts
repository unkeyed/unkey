import type Stripe from "stripe";
import { isDeadSubscription } from "./subscriptionUtils";

export type DeploySubscriptionCancel = "scheduled" | "already-ended";

/**
 * Stops the Stripe renewal for the Deploy subscription, no refund. Deploy owns
 * its own subscription now, so this is a native whole-subscription cancel at
 * period end: usage bills to the boundary, then the plan does not renew. A dead
 * subscription (already cancelled, deleted-webhook lagging) is an idempotent
 * no-op. Lives here rather than in ctrl so Stripe knowledge stays in one place.
 */
export async function cancelDeploySubscription(
  stripe: Stripe,
  subscriptionId: string,
): Promise<DeploySubscriptionCancel> {
  const sub = await stripe.subscriptions.retrieve(subscriptionId);
  if (isDeadSubscription(sub)) {
    return "already-ended";
  }
  await stripe.subscriptions.update(subscriptionId, { cancel_at_period_end: true });
  return "scheduled";
}
