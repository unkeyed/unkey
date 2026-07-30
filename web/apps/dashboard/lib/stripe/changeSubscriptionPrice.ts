import type Stripe from "stripe";

type ChangeSubscriptionPriceInput = {
  subscriptionId: string;
  subscriptionItemId: string;
  newPriceId: string;
  prorationBehavior: Stripe.SubscriptionUpdateParams.ProrationBehavior;
};

export type ChangeSubscriptionPriceResult =
  | { kind: "applied" }
  | { kind: "payment_required"; paymentUrl: string };

/**
 * Changes a price without applying the subscription update until its immediate
 * proration invoice is paid. Stripe's hosted invoice page handles CVC, 3DS, and
 * replacement cards without exposing a raw requires_action error in our UI.
 */
export async function changeSubscriptionPrice(
  stripe: Stripe,
  input: ChangeSubscriptionPriceInput,
): Promise<ChangeSubscriptionPriceResult> {
  const subscription = await stripe.subscriptions.update(input.subscriptionId, {
    items: [{ id: input.subscriptionItemId, price: input.newPriceId }],
    proration_behavior: input.prorationBehavior,
    payment_behavior: "pending_if_incomplete",
    expand: ["latest_invoice"],
  });

  if (!subscription.pending_update) {
    return { kind: "applied" };
  }

  const invoice =
    typeof subscription.latest_invoice === "string"
      ? await stripe.invoices.retrieve(subscription.latest_invoice)
      : subscription.latest_invoice;
  if (!invoice?.hosted_invoice_url) {
    throw new Error("Stripe did not return a payment page for the pending plan change.");
  }

  return { kind: "payment_required", paymentUrl: invoice.hosted_invoice_url };
}
