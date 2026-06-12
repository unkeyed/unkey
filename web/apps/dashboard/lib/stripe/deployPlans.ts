import { stripeEnv } from "@/lib/env";
import { DEPLOY_PLANS, type DeployPlan } from "./deployPlan";

/**
 * Resolved Stripe price ids for Unkey Deploy billing. The webhook detects the
 * plan from price metadata ([[deployPlan]]); these ids are the write side, used
 * by subscribe / change / cancel to attach, swap, and remove the right items.
 */
export type DeployBillingConfig = {
  /** Plan-fee price id per plan. The canonical "has Deploy" item on the sub. */
  planFeePriceIds: Record<DeployPlan, string>;
  /** Metered usage price ids, shared across plans (cpu, memory, egress, disk). */
  meteredPriceIds: string[];
  /** Every Deploy price id (plan-fee + metered), for identifying Deploy items. */
  allDeployPriceIds: Set<string>;
};

/**
 * Reads and validates the Deploy billing price config from env. Returns null if
 * Stripe is not set up or any Deploy price id is missing, so callers can reject
 * with a clear "not configured" error instead of attaching half a subscription.
 */
export function deployBillingConfig(): DeployBillingConfig | null {
  const e = stripeEnv();
  if (!e) {
    return null;
  }

  const planFeePriceIds: Record<DeployPlan, string | undefined> = {
    starter: e.STRIPE_PRICE_DEPLOY_STARTER,
    pro: e.STRIPE_PRICE_DEPLOY_PRO,
    business: e.STRIPE_PRICE_DEPLOY_BUSINESS,
  };
  const meteredPriceIds = [
    e.STRIPE_PRICE_DEPLOY_METER_CPU,
    e.STRIPE_PRICE_DEPLOY_METER_MEMORY,
    e.STRIPE_PRICE_DEPLOY_METER_EGRESS,
    e.STRIPE_PRICE_DEPLOY_METER_DISK,
  ];

  // All-or-nothing: a partially configured set would attach an incomplete
  // subscription (e.g. plan-fee with no metering), so treat it as unconfigured.
  if (DEPLOY_PLANS.some((plan) => !planFeePriceIds[plan]) || meteredPriceIds.some((id) => !id)) {
    return null;
  }

  const resolvedPlanFee = planFeePriceIds as Record<DeployPlan, string>;
  const resolvedMetered = meteredPriceIds as string[];

  return {
    planFeePriceIds: resolvedPlanFee,
    meteredPriceIds: resolvedMetered,
    allDeployPriceIds: new Set<string>([...Object.values(resolvedPlanFee), ...resolvedMetered]),
  };
}

/**
 * The full set of subscription items for a Deploy plan: the plan-fee for the
 * chosen tier plus the shared metered prices. Metered items carry no quantity;
 * usage is billed from meter events.
 */
export function deploySubscriptionItems(
  config: DeployBillingConfig,
  plan: DeployPlan,
): Array<{ price: string }> {
  return [
    { price: config.planFeePriceIds[plan] },
    ...config.meteredPriceIds.map((price) => ({ price })),
  ];
}

/**
 * Maps a plan-fee price id back to its plan, or undefined if it is not a
 * plan-fee price. Used to find the current Deploy plan-fee item on a
 * subscription so change/cancel act on the right item.
 */
export function planForPlanFeePriceId(
  config: DeployBillingConfig,
  priceId: string,
): DeployPlan | undefined {
  return DEPLOY_PLANS.find((plan) => config.planFeePriceIds[plan] === priceId);
}

/** A subscription item, narrowed to the fields the Deploy logic reads. */
export type SubscriptionItemLike = {
  id: string;
  price?: { id?: string | null } | null;
};

/**
 * Returns the Deploy items (plan-fee + metered) among a subscription's items,
 * matched by price id. Used to detect an existing Deploy subscription and to
 * pick which items to remove on cancel.
 */
export function findDeployItems(
  config: DeployBillingConfig,
  items: SubscriptionItemLike[],
): Array<{ id: string; priceId: string }> {
  const found: Array<{ id: string; priceId: string }> = [];
  for (const item of items) {
    const priceId = item.price?.id;
    if (priceId && config.allDeployPriceIds.has(priceId)) {
      found.push({ id: item.id, priceId });
    }
  }
  return found;
}

/**
 * Returns the first subscription item that is not a Deploy item (plan-fee or
 * metered): the API plan item on a mixed subscription. The API routes
 * (create/update/cancel/read) must anchor on this instead of items[0], which
 * on a Compute-first subscription is a Deploy item. When Deploy billing is
 * not configured every item is "not Deploy", which preserves the legacy
 * first-item behavior on single-product subscriptions.
 */
export function findApiItem<T extends SubscriptionItemLike>(
  config: DeployBillingConfig | null,
  items: T[],
): T | undefined {
  if (!config) {
    return items[0];
  }
  return items.find((item) => {
    const priceId = item.price?.id;
    return !priceId || !config.allDeployPriceIds.has(priceId);
  });
}

/**
 * Finds the plan-fee item on a subscription and the plan it maps to, or
 * undefined when no Deploy plan-fee item is present. The plan-fee item is the
 * one change/cancel reprice or anchor on.
 */
export function findPlanFeeItem(
  config: DeployBillingConfig,
  items: SubscriptionItemLike[],
): { id: string; plan: DeployPlan } | undefined {
  for (const item of items) {
    const priceId = item.price?.id;
    if (!priceId) {
      continue;
    }
    const plan = planForPlanFeePriceId(config, priceId);
    if (plan) {
      return { id: item.id, plan };
    }
  }
  return undefined;
}
