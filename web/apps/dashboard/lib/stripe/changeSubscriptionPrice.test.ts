import type Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";
import { changeSubscriptionPrice } from "./changeSubscriptionPrice";

function subscription(overrides: Partial<Stripe.Subscription> = {}): Stripe.Subscription {
  return {
    id: "sub_1",
    pending_update: null,
    latest_invoice: null,
    ...overrides,
  } as unknown as Stripe.Subscription;
}

function invoice(overrides: Partial<Stripe.Invoice> = {}): Stripe.Invoice {
  return {
    id: "in_1",
    hosted_invoice_url: "https://invoice.stripe.test/in_1",
    ...overrides,
  } as unknown as Stripe.Invoice;
}

function pendingUpdate(): Stripe.Subscription.PendingUpdate {
  return {
    billing_cycle_anchor: null,
    discount: null,
    discounts: null,
    expires_at: 1,
    metadata: null,
    subscription_items: null,
    trial_end: null,
    trial_from_plan: null,
  };
}

function stubStripe(updated: Stripe.Subscription, retrievedInvoice = invoice()) {
  const update = vi.fn(async () => updated);
  const retrieveInvoice = vi.fn(async () => retrievedInvoice);
  return {
    stripe: {
      subscriptions: { update },
      invoices: { retrieve: retrieveInvoice },
    } as unknown as Stripe,
    update,
    retrieveInvoice,
  };
}

const input = {
  subscriptionId: "sub_1",
  subscriptionItemId: "si_1",
  newPriceId: "price_new",
  prorationBehavior: "always_invoice" as const,
};

describe("changeSubscriptionPrice", () => {
  it("applies the price through a payment-gated pending update", async () => {
    const { stripe, update } = stubStripe(subscription());

    const result = await changeSubscriptionPrice(stripe, input);

    expect(result).toEqual({ kind: "applied" });
    expect(update).toHaveBeenCalledWith("sub_1", {
      items: [{ id: "si_1", price: "price_new" }],
      proration_behavior: "always_invoice",
      payment_behavior: "pending_if_incomplete",
      expand: ["latest_invoice"],
    });
  });

  it("returns the expanded hosted invoice when payment needs customer action", async () => {
    const hostedInvoice = invoice();
    const { stripe, retrieveInvoice } = stubStripe(
      subscription({ pending_update: pendingUpdate(), latest_invoice: hostedInvoice }),
    );

    const result = await changeSubscriptionPrice(stripe, input);

    expect(result).toEqual({
      kind: "payment_required",
      paymentUrl: "https://invoice.stripe.test/in_1",
    });
    expect(retrieveInvoice).not.toHaveBeenCalled();
  });

  it("retrieves the hosted invoice when Stripe returns only its id", async () => {
    const { stripe, retrieveInvoice } = stubStripe(
      subscription({ pending_update: pendingUpdate(), latest_invoice: "in_1" }),
    );

    const result = await changeSubscriptionPrice(stripe, input);

    expect(result.kind).toBe("payment_required");
    expect(retrieveInvoice).toHaveBeenCalledWith("in_1");
  });

  it("fails instead of applying entitlement when Stripe has no payment page", async () => {
    const { stripe } = stubStripe(
      subscription({
        pending_update: pendingUpdate(),
        latest_invoice: invoice({ hosted_invoice_url: null }),
      }),
    );

    await expect(changeSubscriptionPrice(stripe, input)).rejects.toThrow(
      "Stripe did not return a payment page for the pending plan change.",
    );
  });
});
