import { createHash } from "node:crypto";
import type Stripe from "stripe";
import type { SubscriptionProduct } from "./billingSubscriptions";
import { isDeadSubscription } from "./subscriptionUtils";

type SubscriptionCheckoutInput = {
  workspaceId: string;
  product: SubscriptionProduct;
  customerId?: string;
  lineItems: NonNullable<Stripe.Checkout.SessionCreateParams["line_items"]>;
  successUrl: string;
  customText?: Stripe.Checkout.SessionCreateParams.CustomText;
  idempotencyKey?: string;
};

export type SubscriptionCheckoutDestination =
  | { kind: "checkout"; session: Stripe.Checkout.Session }
  | { kind: "success"; url: string };

/**
 * Creates the first subscription for either paid product in Checkout. Keeping
 * the customer, payment behavior, and stale-session recovery here prevents one
 * product from bypassing Checkout when a saved card needs CVC or 3DS.
 */
export async function createSubscriptionCheckout(
  stripe: Stripe,
  input: SubscriptionCheckoutInput,
): Promise<SubscriptionCheckoutDestination> {
  const sessionParams: Stripe.Checkout.SessionCreateParams = {
    client_reference_id: input.workspaceId,
    metadata: { unkey_product: input.product },
    billing_address_collection: "auto",
    mode: "subscription",
    line_items: input.lineItems,
    // Checkout hides `limited` and legacy `unspecified` cards by default even
    // when they are the customer's invoice default. These cards were already
    // collected for workspace billing, so show them for either paid product.
    saved_payment_method_options: {
      allow_redisplay_filters: ["always", "limited", "unspecified"],
    },
    subscription_data: {
      metadata: { unkey_product: input.product, workspace_id: input.workspaceId },
      billing_cycle_anchor_config: { day_of_month: 1, hour: 0, minute: 0, second: 0 },
      billing_mode: { type: "classic" },
      proration_behavior: "create_prorations",
    },
    ...(input.customerId ? { customer: input.customerId } : {}),
    ...(input.customText ? { custom_text: input.customText } : {}),
    success_url: input.successUrl,
  };

  if (!input.idempotencyKey) {
    return {
      kind: "checkout",
      session: await stripe.checkout.sessions.create(sessionParams),
    };
  }

  // Stripe replays the creation-time response for an idempotency key, so an
  // old session can still look open in that response after it was completed or
  // expired. Re-read it: resume a genuinely open session, send a paid live
  // subscription through /success for linking, or derive a fresh deterministic
  // key from the stale session instead of creating duplicates on retries.
  // Include a compact request fingerprint so catalog prices, return URLs, or
  // custom text can change without Stripe rejecting the reused base key because
  // its parameters differ from the original request.
  const requestFingerprint = createHash("sha256")
    .update(JSON.stringify(sessionParams))
    .digest("hex")
    .slice(0, 16);
  let idempotencyKey = `${input.idempotencyKey}:${requestFingerprint}`;
  let session = await stripe.checkout.sessions.create(sessionParams, { idempotencyKey });
  for (let attempt = 0; attempt < 3; attempt++) {
    const live = await stripe.checkout.sessions.retrieve(session.id);
    if (live.status === "open") {
      return { kind: "checkout", session: live };
    }
    if (live.status === "complete") {
      const subscriptionId =
        typeof live.subscription === "string" ? live.subscription : live.subscription?.id;
      const subscription = subscriptionId
        ? await stripe.subscriptions.retrieve(subscriptionId)
        : null;
      if (subscription && !isDeadSubscription(subscription)) {
        return {
          kind: "success",
          url: input.successUrl.replace("{CHECKOUT_SESSION_ID}", live.id),
        };
      }
    }

    idempotencyKey = `${idempotencyKey}:${session.id}`;
    session = await stripe.checkout.sessions.create(sessionParams, { idempotencyKey });
  }

  return { kind: "checkout", session };
}
