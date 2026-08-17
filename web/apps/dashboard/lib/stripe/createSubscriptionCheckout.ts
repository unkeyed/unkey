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

/** Creates a payment-capable first-subscription Checkout for API or Compute. */
export async function createSubscriptionCheckout(
  stripe: Stripe,
  input: SubscriptionCheckoutInput,
): Promise<SubscriptionCheckoutDestination> {
  const sessionParams: Stripe.Checkout.SessionCreateParams = {
    client_reference_id: input.workspaceId,
    metadata: { unkey_product: input.product },
    billing_address_collection: "auto",
    mode: "subscription",
    payment_method_types: ["card"],
    line_items: input.lineItems,
    // Cards collected by older setup flows can be limited or unspecified. They
    // remain valid workspace methods and should be shown alongside a new-card
    // option when the customer needs to replace one.
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

  // Stripe replays the creation-time response for an idempotency key. Re-read
  // it so an actually-open session is reused, a paid one returns through the
  // linker, and an expired/dead one gets a fresh deterministic key. Fingerprint
  // all request parameters so changing the customer or catalog does not collide
  // with an older request under the same logical base key.
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

    const retryFingerprint = createHash("sha256")
      .update(`${idempotencyKey}:${session.id}`)
      .digest("hex")
      .slice(0, 16);
    idempotencyKey = `${input.idempotencyKey}:${requestFingerprint}:retry:${retryFingerprint}`;
    session = await stripe.checkout.sessions.create(sessionParams, { idempotencyKey });
  }

  return { kind: "checkout", session };
}
