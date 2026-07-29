import type Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import { createSubscriptionCheckout } from "./createSubscriptionCheckout";

function checkoutSession(
  overrides: Partial<Stripe.Checkout.Session> = {},
): Stripe.Checkout.Session {
  return {
    id: "cs_1",
    status: "open",
    url: "https://checkout.stripe.test/cs_1",
    subscription: null,
    ...overrides,
  } as unknown as Stripe.Checkout.Session;
}

function subscription(status: Stripe.Subscription.Status): Stripe.Subscription {
  return { id: "sub_1", status } as unknown as Stripe.Subscription;
}

function stubStripe(input?: {
  created?: Stripe.Checkout.Session[];
  retrieved?: Stripe.Checkout.Session[];
  subscription?: Stripe.Subscription;
}) {
  const created = input?.created ?? [checkoutSession()];
  const retrieved = input?.retrieved ?? [checkoutSession()];
  const create = vi.fn(
    async (_params: Stripe.Checkout.SessionCreateParams, _options?: Stripe.RequestOptions) =>
      created.shift() ?? checkoutSession(),
  );
  const retrieveSession = vi.fn(async (_id: string) => retrieved.shift() ?? checkoutSession());
  const retrieveSubscription = vi.fn(
    async (_id: string) => input?.subscription ?? subscription("active"),
  );

  return {
    stripe: {
      checkout: { sessions: { create, retrieve: retrieveSession } },
      subscriptions: { retrieve: retrieveSubscription },
    } as unknown as Stripe,
    create,
    retrieveSession,
    retrieveSubscription,
  };
}

const baseInput = {
  workspaceId: "ws_1",
  customerId: "cus_1",
  lineItems: [{ price: "price_1", quantity: 1 }],
  successUrl: "https://app.test/success?session_id={CHECKOUT_SESSION_ID}",
};

describe("createSubscriptionCheckout", () => {
  it.each(["api", "compute"] as const)(
    "uses the payment-capable session shape for %s",
    async (product) => {
      const { stripe, create, retrieveSession } = stubStripe();

      const result = await createSubscriptionCheckout(stripe, { ...baseInput, product });

      expect(result.kind).toBe("checkout");
      expect(retrieveSession).not.toHaveBeenCalled();
      expect(create).toHaveBeenCalledWith({
        client_reference_id: "ws_1",
        metadata: { unkey_product: product },
        billing_address_collection: "auto",
        mode: "subscription",
        payment_method_types: ["card"],
        customer: "cus_1",
        line_items: [{ price: "price_1", quantity: 1 }],
        saved_payment_method_options: {
          allow_redisplay_filters: ["always", "limited", "unspecified"],
        },
        subscription_data: {
          metadata: { unkey_product: product, workspace_id: "ws_1" },
          billing_cycle_anchor_config: { day_of_month: 1, hour: 0, minute: 0, second: 0 },
          billing_mode: { type: "classic" },
          proration_behavior: "create_prorations",
        },
        success_url: baseInput.successUrl,
      });
    },
  );

  it("reuses an open idempotent session", async () => {
    const live = checkoutSession({ id: "cs_open", url: "https://checkout.stripe.test/open" });
    const { stripe, create } = stubStripe({ retrieved: [live] });

    const result = await createSubscriptionCheckout(stripe, {
      ...baseInput,
      product: "api",
      idempotencyKey: "api-checkout:ws_1:prod_1",
    });

    expect(result).toEqual({ kind: "checkout", session: live });
    expect(create.mock.calls[0]?.[1]?.idempotencyKey).toMatch(
      /^api-checkout:ws_1:prod_1:[a-f0-9]{16}$/,
    );
    expect(create).toHaveBeenCalledTimes(1);
  });

  it("changes the idempotency key when Checkout parameters change", async () => {
    const first = stubStripe();
    const second = stubStripe();

    await createSubscriptionCheckout(first.stripe, {
      ...baseInput,
      product: "compute",
      idempotencyKey: "deploy-checkout:ws_1:starter",
    });
    await createSubscriptionCheckout(second.stripe, {
      ...baseInput,
      product: "compute",
      customText: { submit: { message: "Credits match this charge." } },
      idempotencyKey: "deploy-checkout:ws_1:starter",
    });

    expect(first.create.mock.calls[0]?.[1]?.idempotencyKey).not.toBe(
      second.create.mock.calls[0]?.[1]?.idempotencyKey,
    );
  });

  it("routes a completed session with a live subscription through success", async () => {
    const paid = checkoutSession({
      id: "cs_paid",
      status: "complete",
      subscription: "sub_1",
    });
    const { stripe, retrieveSubscription } = stubStripe({
      retrieved: [paid],
      subscription: subscription("active"),
    });

    const result = await createSubscriptionCheckout(stripe, {
      ...baseInput,
      product: "compute",
      idempotencyKey: "deploy-checkout:ws_1:starter",
    });

    expect(result).toEqual({
      kind: "success",
      url: "https://app.test/success?session_id=cs_paid",
    });
    expect(retrieveSubscription).toHaveBeenCalledWith("sub_1");
  });

  it("chains a fresh key after a completed session whose subscription is dead", async () => {
    const stale = checkoutSession({
      id: "cs_stale",
      status: "complete",
      subscription: "sub_1",
    });
    const fresh = checkoutSession({ id: "cs_fresh" });
    const { stripe, create } = stubStripe({
      created: [checkoutSession({ id: "cs_stale" }), fresh],
      retrieved: [stale, fresh],
      subscription: subscription("canceled"),
    });

    const result = await createSubscriptionCheckout(stripe, {
      ...baseInput,
      product: "compute",
      idempotencyKey: "deploy-checkout:ws_1:starter",
    });

    expect(result).toEqual({ kind: "checkout", session: fresh });
    const initialKey = create.mock.calls[0]?.[1]?.idempotencyKey;
    expect(create.mock.calls[1]?.[1]?.idempotencyKey).toMatch(
      /^deploy-checkout:ws_1:starter:[a-f0-9]{16}:retry:[a-f0-9]{16}$/,
    );
    expect(create.mock.calls[1]?.[1]?.idempotencyKey).not.toBe(initialKey);
  });
});
