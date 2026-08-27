import type { SubscriptionProduct } from "./billingSubscriptions";
import { deployPlanGrantsTeam } from "./deployPlan";

type StripeWebhookResponseDetails = Record<string, string | number | boolean | null>;

/**
 * Returns a Stripe-visible receipt with enough context to explain how an event
 * was handled. Stripe's delivery log shows both the JSON body and these headers,
 * so an acknowledged no-op is distinguishable from an alert that was attempted.
 */
export function stripeWebhookResponse(
  event: { id: string; type: string },
  result: string,
  details: StripeWebhookResponseDetails = {},
): Response {
  return Response.json(
    {
      eventId: event.id,
      eventType: event.type,
      result,
      details,
    },
    {
      status: 200,
      headers: {
        "X-Unkey-Stripe-Event-Id": event.id,
        "X-Unkey-Stripe-Event-Type": event.type,
        "X-Unkey-Webhook-Result": result,
      },
    },
  );
}

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
