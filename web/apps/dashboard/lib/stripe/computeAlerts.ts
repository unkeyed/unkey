import type Stripe from "stripe";
import { formatPrice } from "@/lib/fmt";
import type { DeployBillingConfig } from "./deployBilling";
import { planForPlanFeePriceId } from "./deployBilling";
import { DEPLOY_PLANS, type DeployPlan, detectDeployPlan } from "./deployPlan";

/**
 * The operational Slack alert a Compute (Deploy) subscription lifecycle event
 * should raise, or null for events that warrant none (renewals, metered usage,
 * card updates). Each variant carries exactly the strings its slack alert needs;
 * the caller adds the customer email/name it resolves from Stripe.
 *
 * These are pure functions on the Stripe event so they stay unit-testable: the
 * dashboard mutations write workspace_billing.plan optimistically before Stripe
 * delivers the webhook, so a DB plan diff always looks unchanged and cannot be
 * the signal. The signal is the Stripe event itself.
 */
export type ComputeLifecycleAlert =
  | { type: "created"; product: string; price: string }
  | { type: "cancelling"; product: string; price: string }
  | {
      type: "updated";
      product: string;
      price: string;
      changeType: "upgraded" | "downgraded";
      previousTier: string;
    };

/** Title-cases a Compute plan for display, e.g. "pro" -> "Compute Pro". */
export function computeProductLabel(plan: DeployPlan): string {
  return `Compute ${plan.charAt(0).toUpperCase()}${plan.slice(1)}`;
}

/**
 * The recurring plan-fee amount for a Compute plan, formatted as a price string.
 * The plan-fee is the licensed item tagged with metadata plan=<plan> (the same
 * marker detectDeployPlan keys on); the shared metered items carry no fixed
 * amount. Falls back to "n/a" when the amount is absent, which only happens on a
 * malformed price and must not stop the alert.
 */
function planFeePrice(sub: Stripe.Subscription, plan: DeployPlan): string {
  const item = sub.items?.data?.find((i) => i.price?.metadata?.plan?.trim() === plan);
  const amount = item?.price?.unit_amount;
  return typeof amount === "number" ? formatPrice(amount) : "n/a";
}

/** Upgrade vs downgrade from the plan ordering, or null when the plan is unchanged. */
function planDirection(from: DeployPlan, to: DeployPlan): "upgraded" | "downgraded" | null {
  const fromIndex = DEPLOY_PLANS.indexOf(from);
  const toIndex = DEPLOY_PLANS.indexOf(to);
  if (toIndex > fromIndex) {
    return "upgraded";
  }
  if (toIndex < fromIndex) {
    return "downgraded";
  }
  return null;
}

/**
 * The plan a subscription carried before an update, read from the previous
 * plan-fee item in the event's previous_attributes. Stripe includes the prior
 * items list only when the items actually changed (a changeDeployPlan reprice),
 * so a present list is the signal a plan swap occurred; the price id maps back to
 * a plan through the billing config. Returns null when the config is unavailable
 * or no prior item resolves to a plan.
 */
function previousPlanFromItems(
  config: DeployBillingConfig,
  previous: Partial<Stripe.Subscription> | undefined,
): DeployPlan | null {
  for (const item of previous?.items?.data ?? []) {
    const priceId = item.price?.id;
    if (!priceId) {
      continue;
    }
    const plan = planForPlanFeePriceId(config, priceId);
    if (plan) {
      return plan;
    }
  }
  return null;
}

/**
 * The alert for a Compute customer.subscription.created event: a created
 * subscription carrying a recognized Compute plan is a new subscription. Returns
 * null when the subscription carries no recognized Compute plan (nothing to
 * announce).
 */
export function computeCreatedAlert(sub: Stripe.Subscription): ComputeLifecycleAlert | null {
  const plan = detectDeployPlan(sub);
  if (!plan) {
    return null;
  }
  return { type: "created", product: computeProductLabel(plan), price: planFeePrice(sub, plan) };
}

/**
 * The alert for a Compute customer.subscription.updated event.
 *
 * - A cancel just scheduled (cancel_at_period_end / cancel_at newly set) raises a
 *   "cancelling" alert. The plan-fee item is still on the subscription during a
 *   scheduled cancel, so its plan and price are read from the live subscription.
 *   Gating on the field appearing in previous_attributes stops an unrelated
 *   update on an already-cancelling subscription from re-alerting.
 * - A plan swap (previous_attributes carries the prior items) raises an
 *   "upgraded"/"downgraded" alert, direction from the plan ordering.
 * - Everything else (renewals, metered usage, card updates, resumes) returns
 *   null: those do not change items or the cancel fields.
 */
export function computeUpdatedAlert(
  config: DeployBillingConfig | null,
  sub: Stripe.Subscription,
  previousAttributes: Partial<Stripe.Subscription> | undefined,
): ComputeLifecycleAlert | null {
  const currentPlan = detectDeployPlan(sub);

  const cancelling = Boolean(sub.cancel_at_period_end) || Boolean(sub.cancel_at);
  const cancelJustSet =
    previousAttributes != null &&
    ("cancel_at_period_end" in previousAttributes || "cancel_at" in previousAttributes);
  if (cancelling && cancelJustSet && currentPlan) {
    return {
      type: "cancelling",
      product: computeProductLabel(currentPlan),
      price: planFeePrice(sub, currentPlan),
    };
  }

  // A plan swap is the only reason a Compute subscription's items change, so a
  // present prior-items list means the plan-fee was repriced. Resolve both ends
  // and only alert when they genuinely differ.
  if (previousAttributes?.items && currentPlan && config) {
    const previousPlan = previousPlanFromItems(config, previousAttributes);
    if (previousPlan && previousPlan !== currentPlan) {
      const changeType = planDirection(previousPlan, currentPlan);
      if (changeType) {
        return {
          type: "updated",
          product: computeProductLabel(currentPlan),
          price: planFeePrice(sub, currentPlan),
          changeType,
          previousTier: computeProductLabel(previousPlan),
        };
      }
    }
  }

  return null;
}
