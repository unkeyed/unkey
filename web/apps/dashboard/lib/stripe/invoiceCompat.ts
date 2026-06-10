import type Stripe from "stripe";
import { z } from "zod";

/**
 * Dual-shape readers for Stripe objects that can arrive in either the modern
 * (basil+) or a legacy (pre-2025-03-31) shape.
 *
 * SDK-fetched objects are always modern, but webhook payloads follow the
 * webhook ENDPOINT's pinned API version, not the SDK's. Until the endpoint is
 * bumped to 2026-05-27.dahlia (and for events replayed from before the bump),
 * both shapes are live in the webhook path. The legacy fields no longer exist
 * on the SDK types, so they are parsed out with zod schemas at the boundary,
 * never by casting to old types.
 *
 * Once the endpoint version is bumped everywhere, the legacy halves of these
 * helpers can be deleted.
 */

/** An expandable Stripe reference: an id string, or the expanded object. */
const expandableId = z
  .union([z.string(), z.object({ id: z.string() })])
  .transform((ref) => (typeof ref === "string" ? ref : ref.id));

// Every legacy field carries .catch(undefined): on modern payloads the field
// is absent (or reshaped), and these readers must degrade to "not present"
// instead of throwing.
const legacyInvoice = z.object({
  subscription: expandableId.nullish().catch(undefined),
  payment_intent: expandableId.nullish().catch(undefined),
});

const legacyLine = z.object({
  price: z.object({ id: z.string() }).nullish().catch(undefined),
  proration: z.boolean().nullish().catch(undefined),
});

const legacySubscription = z.object({
  current_period_start: z.number().nullish().catch(undefined),
});

/**
 * The subscription an invoice belongs to. Modern:
 * invoice.parent.subscription_details.subscription; legacy:
 * invoice.subscription.
 */
export function invoiceSubscriptionId(invoice: Stripe.Invoice): string | undefined {
  if (invoice.parent?.type === "subscription_details") {
    const subscription = invoice.parent.subscription_details?.subscription;
    if (subscription) {
      return typeof subscription === "string" ? subscription : subscription.id;
    }
  }
  return legacyInvoice.parse(invoice).subscription ?? undefined;
}

/**
 * The payment intent behind an invoice. Modern: the payments list (first
 * payment-intent entry); legacy: invoice.payment_intent.
 */
export function invoicePaymentIntentId(invoice: Stripe.Invoice): string | undefined {
  for (const { payment } of invoice.payments?.data ?? []) {
    if (payment.type === "payment_intent" && payment.payment_intent) {
      const intent = payment.payment_intent;
      return typeof intent === "string" ? intent : intent.id;
    }
  }
  return legacyInvoice.parse(invoice).payment_intent ?? undefined;
}

/**
 * The price id of an invoice line. Modern: line.pricing.price_details.price
 * (an id string, never expanded); legacy: line.price.id.
 */
export function invoiceLinePriceId(line: Stripe.InvoiceLineItem): string | undefined {
  const modern = line.pricing?.price_details?.price;
  if (typeof modern === "string") {
    return modern;
  }
  return legacyLine.parse(line).price?.id;
}

/**
 * Whether an invoice line is a proration. Modern:
 * line.parent.subscription_item_details.proration; legacy: line.proration.
 */
export function invoiceLineIsProration(line: Stripe.InvoiceLineItem): boolean {
  if (line.parent?.type === "subscription_item_details") {
    return Boolean(line.parent.subscription_item_details?.proration);
  }
  return legacyLine.parse(line).proration === true;
}

/**
 * The start of a subscription's current billing period. Modern: per-item
 * (earliest across items); legacy: top-level current_period_start.
 */
export function subscriptionCurrentPeriodStart(sub: Stripe.Subscription): number | undefined {
  const starts = (sub.items?.data ?? [])
    .map((item) => item.current_period_start)
    .filter((start): start is number => typeof start === "number");
  if (starts.length > 0) {
    return Math.min(...starts);
  }
  return legacySubscription.parse(sub).current_period_start ?? undefined;
}
