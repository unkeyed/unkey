import type Stripe from "stripe";

/**
 * The Unkey Deploy plans we recognize, lowest to highest. Mirrored into
 * workspaces.deploy_plan; NULL (absence) means no Deploy plan.
 *
 * Adding a plan: create its Stripe price(s) tagged with metadata
 * `deploy_plan=<name>` and add the name here. No price ids are hardcoded, so
 * re-pricing an existing plan needs no change at all: the new price carries the
 * same metadata and existing subscriptions keep resolving (grandfathered).
 */
export const DEPLOY_PLANS = ["starter", "pro", "business"] as const;
export type DeployPlan = (typeof DEPLOY_PLANS)[number];

/** Stripe price metadata key carrying the Deploy plan name. */
const DEPLOY_PLAN_METADATA_KEY = "deploy_plan";

function isDeployPlan(value: string): value is DeployPlan {
  return (DEPLOY_PLANS as readonly string[]).includes(value);
}

/**
 * Detects which Deploy plan, if any, a Stripe subscription carries.
 *
 * The Deploy plan-fee price is tagged in Stripe with metadata
 * `deploy_plan=<plan>`, the canonical "has Deploy" marker. We scan the
 * subscription's items (the plan-fee sits alongside API items and metered Deploy
 * prices) and return the first recognized plan. price.metadata ships in the
 * webhook payload, so this needs no extra Stripe calls.
 *
 * Fails closed: an item tagged with an unrecognized plan is logged and ignored,
 * so a Stripe typo can never grant Deploy access. Returns null when no item
 * carries a recognized Deploy plan.
 */
export function detectDeployPlan(sub: Stripe.Subscription): DeployPlan | null {
  for (const item of sub.items?.data ?? []) {
    const raw = item.price?.metadata?.[DEPLOY_PLAN_METADATA_KEY]?.trim();
    if (!raw) {
      continue;
    }
    if (isDeployPlan(raw)) {
      return raw;
    }
    console.warn("Subscription item carries an unrecognized Deploy plan metadata value", {
      subscriptionId: sub.id,
      priceId: item.price?.id,
      deployPlan: raw,
    });
  }
  return null;
}
