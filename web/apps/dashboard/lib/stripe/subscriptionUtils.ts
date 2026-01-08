import type Stripe from "stripe";

interface PreviousAttributes {
  // Billing period dates (change during automated renewals)
  current_period_end?: number;
  current_period_start?: number;

  // Subscription items and pricing (change during manual updates)
  items?: {
    data?: Partial<Stripe.SubscriptionItem>[];
  };

  // Other subscription properties that can change manually
  plan?: Stripe.Plan | null;
  quantity?: number;
  discount?: Stripe.Discount | null;
  cancel_at_period_end?: boolean;
  collection_method?: string;
  latest_invoice?: string | Stripe.Invoice | null;

  // Payment method changes (when users update their card)
  default_payment_method?: string | Stripe.PaymentMethod | null;

  // Status changes (can indicate payment failures)
  status?: Stripe.Subscription.Status;
}

/**
 * Determines if a subscription update is related to payment failure.
 * This happens when:
 * 1. Subscription status changed to past_due, unpaid, or incomplete
 * 2. Latest invoice changed (indicating a payment attempt)
 * 3. No manual changes to pricing, plan, or other subscription settings
 */
export function isPaymentFailureRelatedUpdate(
  sub: Stripe.Subscription,
  previousAttributes: PreviousAttributes | undefined,
): boolean {
  if (!previousAttributes) {
    return false;
  }

  const changedKeys = Object.keys(previousAttributes);

  // Check if status changed to a payment-failure-related status
  const paymentFailureStatuses = ["past_due", "unpaid", "incomplete"];
  const statusChanged =
    changedKeys.includes("status") && paymentFailureStatuses.includes(sub.status);

  // Check if latest_invoice changed (indicates payment processing)
  const invoiceChanged = changedKeys.includes("latest_invoice");

  // Define keys that indicate manual changes (not payment-related)
  const manualChangeKeys = [
    "cancel_at_period_end",
    "collection_method",
    "plan",
    "quantity",
    "discount",
    "items", // pricing/plan changes
  ];

  // If any manual change keys are present, this is not a payment failure update
  const hasManualChanges = manualChangeKeys.some((key) => changedKeys.includes(key));

  // Consider it a payment failure update if:
  // - Status changed to payment failure status, OR
  // - Latest invoice changed without manual subscription changes
  return (statusChanged || invoiceChanged) && !hasManualChanges;
}

/**
 * Determines if a subscription update is an automated billing renewal.
 * Treat as automated renewal when:
 * 1. subscription status is active
 * 2. previousAttributes exists
 * 3. Only contains billing period changes (current_period_start, current_period_end) and optionally items/latest_invoice
 * 4. If items changed, only the period dates within items actually changed (not price/plan/quantity)
 * 5. cancel_at_period_end and collection_method are not present among keys
 */
export function isAutomatedBillingRenewal(
  sub: Stripe.Subscription,
  previousAttributes: PreviousAttributes | undefined,
): boolean {
  if (sub.status !== "active" || !previousAttributes) {
    return false;
  }

  // Get all keys that changed in previousAttributes
  const changedKeys = Object.keys(previousAttributes);

  // Define keys that indicate manual changes (not automated renewals)
  const manualChangeKeys = [
    "cancel_at_period_end",
    "collection_method",
    "plan",
    "quantity",
    "discount",
  ];

  // If any manual change keys are present, this is not an automated renewal
  if (manualChangeKeys.some((key) => changedKeys.includes(key))) {
    return false;
  }

  // Check if items changed and verify only period dates changed
  if (changedKeys.includes("items")) {
    const itemsChange = previousAttributes.items;
    if (!itemsChange || !itemsChange.data || !itemsChange.data[0] || !sub.items?.data?.[0]) {
      return false;
    }

    const previousItem = itemsChange.data[0];
    const currentItem = sub.items.data[0];

    // Check if price, plan, or quantity actually changed by comparing current vs previous
    if (
      previousItem.price?.id !== currentItem.price?.id ||
      previousItem.plan?.id !== currentItem.plan?.id ||
      previousItem.quantity !== currentItem.quantity
    ) {
      return false;
    }
  }

  // Define expected keys for automated renewal (period dates + optional items/latest_invoice)
  const allowedKeys = ["current_period_start", "current_period_end", "items", "latest_invoice"];

  // Check if all changed keys are allowed for automated renewals
  const hasOnlyAllowedKeys = changedKeys.every((key) => allowedKeys.includes(key));

  return hasOnlyAllowedKeys;
}

/**
 * Determines if a subscription update is only a payment method (card) update.
 * This happens when:
 * 1. Only the default_payment_method field changed
 * 2. No other subscription properties changed (pricing, plan, status, etc.)
 */
export function isCardUpdateOnly(
  _sub: Stripe.Subscription,
  previousAttributes: PreviousAttributes | undefined,
): boolean {
  if (!previousAttributes) {
    return false;
  }

  const changedKeys = Object.keys(previousAttributes);

  // Check if only default_payment_method changed
  if (changedKeys.length === 1 && changedKeys.includes("default_payment_method")) {
    return true;
  }

  return false;
}

export type { PreviousAttributes };
