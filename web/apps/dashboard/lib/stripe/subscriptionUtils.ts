import type Stripe from "stripe";

interface PreviousAttributes {
  // Billing period dates (change during automated renewals). Since
  // Stripe API 2025-03-31 (basil) these arrive per-item under
  // items.data[].current_period_*; the top-level fields cover replayed
  // events from before the endpoint version bump.
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

    // previous_attributes carries only the fields that changed, so a key
    // that is absent means "unchanged" — only compare keys that are present.
    // Basil+ webhook endpoints report renewals as items.data[] diffs carrying
    // just current_period_*; treating their missing price as a change would
    // misclassify every renewal as a manual update.
    if (
      ("price" in previousItem && previousItem.price?.id !== currentItem.price?.id) ||
      ("plan" in previousItem && previousItem.plan?.id !== currentItem.plan?.id) ||
      ("quantity" in previousItem && previousItem.quantity !== currentItem.quantity)
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

/**
 * A subscription that is over and can never bill again. workspaces.
 * stripe_subscription_id can point at one of these: cancelDeploy cancels a
 * Compute-only subscription outright, and the customer.subscription.deleted
 * webhook that clears the column may lag or be missed. Callers that gate on
 * "workspace already has a subscription" must treat a dead one as absent, or
 * a workspace that cancels mid-month can never subscribe again.
 */
export function isDeadSubscription(sub: Stripe.Subscription): boolean {
  return sub.status === "canceled" || sub.status === "incomplete_expired";
}

/**
 * Metadata marker on subscription schedules created by cancelSubscription to
 * phase the API plan off a mixed (API + Compute) subscription at period end —
 * Stripe has no per-item cancel_at_period_end, so a schedule stands in for it.
 * uncancelSubscription releases marked schedules to resume the plan,
 * getBillingInfo reports their phase boundary as cancelAt, and the
 * subscription.updated webhook downgrades the workspace when the boundary hits.
 */
export const API_CANCEL_SCHEDULE_MARKER = "unkey_api_cancel_at_period_end";

/**
 * The API-plan cancellation schedule managing this subscription, or null when
 * the subscription is unmanaged or managed by some other schedule (which the
 * cancel flows must not touch).
 */
export async function getApiCancelSchedule(
  stripe: Stripe,
  sub: Stripe.Subscription,
): Promise<Stripe.SubscriptionSchedule | null> {
  if (!sub.schedule) {
    return null;
  }
  const schedule =
    typeof sub.schedule === "string"
      ? await stripe.subscriptionSchedules.retrieve(sub.schedule)
      : sub.schedule;
  return schedule.metadata?.[API_CANCEL_SCHEDULE_MARKER] === "true" ? schedule : null;
}

/**
 * Whether a schedule is still governing upcoming phases (releasing or
 * composing with it only makes sense then).
 */
export function isPendingSchedule(schedule: Stripe.SubscriptionSchedule): boolean {
  return schedule.status === "active" || schedule.status === "not_started";
}

/**
 * Creates the schedule that phases the API items off a mixed subscription at
 * period end: the current items run to the boundary, then one Compute-only
 * iteration, then end_behavior "release" detaches the schedule and the
 * subscription returns to normal item-level management. Marked with
 * API_CANCEL_SCHEDULE_MARKER so the other billing flows can recognize (and
 * compose with) the pending cancellation.
 */
export async function createApiCancelSchedule(
  stripe: Stripe,
  subscriptionId: string,
  deployItems: Stripe.SubscriptionItem[],
): Promise<void> {
  const schedule = await stripe.subscriptionSchedules.create({
    from_subscription: subscriptionId,
  });
  const currentPhase = schedule.phases[0];

  const phaseItems = (
    items: Array<{ price: string | Stripe.Price | Stripe.DeletedPrice; quantity?: number }>,
  ): Stripe.SubscriptionScheduleUpdateParams.Phase.Item[] =>
    items.map((item) => ({
      price: typeof item.price === "string" ? item.price : item.price.id,
      // Metered prices must not carry a quantity; licensed ones keep theirs.
      ...(item.quantity ? { quantity: item.quantity } : {}),
    }));

  await stripe.subscriptionSchedules.update(schedule.id, {
    metadata: { [API_CANCEL_SCHEDULE_MARKER]: "true" },
    end_behavior: "release",
    phases: [
      // The current phase must be passed back unchanged (Stripe requires
      // restating in-progress phases on every update).
      {
        items: phaseItems(currentPhase.items),
        start_date: currentPhase.start_date,
        end_date: currentPhase.end_date,
      },
      {
        items: phaseItems(deployItems),
        // One billing month of the Compute-only phase, then release. Compute
        // is anchored to the 1st, so this spans exactly one billing period.
        duration: { interval: "month" },
        // The API item ran its full paid period, so there is nothing to
        // credit at the boundary.
        proration_behavior: "none",
      },
    ],
  });
}
