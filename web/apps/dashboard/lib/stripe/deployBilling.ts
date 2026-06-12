import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
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

// Resolved lookup_keys -> price ids change only on a reprice, which is rare, so
// the resolution is cached: long enough to avoid a prices.list call on every
// request, short enough to pick up a reprice without a redeploy.
const RESOLUTION_TTL_MS = 5 * 60 * 1000;
let cache: { key: string; expiresAt: number; config: DeployBillingConfig } | null = null;

/**
 * Whether every Deploy lookup_key env var is set. deployBillingConfig() returns
 * null both when Deploy is unconfigured (these keys absent) AND when configured
 * keys fail to resolve to active prices (a transient reprice / Stripe window).
 * Callers that make a destructive choice from item classification use this to
 * tell the two apart: unconfigured means "not a Deploy subscription" (treating
 * the first item as the API item is safe), while configured-but-unresolved means
 * they must fail closed rather than guess which item to touch.
 */
export function deployBillingConfigured(): boolean {
  const e = stripeEnv();
  if (!e) {
    return false;
  }
  return [
    e.STRIPE_LOOKUP_DEPLOY_STARTER,
    e.STRIPE_LOOKUP_DEPLOY_PRO,
    e.STRIPE_LOOKUP_DEPLOY_BUSINESS,
    e.STRIPE_LOOKUP_DEPLOY_METER_CPU,
    e.STRIPE_LOOKUP_DEPLOY_METER_MEMORY,
    e.STRIPE_LOOKUP_DEPLOY_METER_EGRESS,
    e.STRIPE_LOOKUP_DEPLOY_METER_DISK,
    e.STRIPE_LOOKUP_DEPLOY_METER_ACTIVE_KEYS,
  ].every(Boolean);
}

/**
 * Resolves and validates the Deploy billing config. Env holds Stripe price
 * lookup_keys (stable handles); this resolves them to the current active price
 * ids via one prices.list call, cached. A reprice transfers the lookup_key onto
 * the new price, so the ids update with no env change. Returns null if Stripe is
 * not configured or any lookup_key is missing or unresolved, so callers can
 * reject with a clear "not configured" error instead of attaching half a
 * subscription.
 */
export async function deployBillingConfig(): Promise<DeployBillingConfig | null> {
  const e = stripeEnv();
  if (!e) {
    return null;
  }

  const planFeeLookupKeys: Record<DeployPlan, string | undefined> = {
    starter: e.STRIPE_LOOKUP_DEPLOY_STARTER,
    pro: e.STRIPE_LOOKUP_DEPLOY_PRO,
    business: e.STRIPE_LOOKUP_DEPLOY_BUSINESS,
  };
  const meterLookupKeys = [
    e.STRIPE_LOOKUP_DEPLOY_METER_CPU,
    e.STRIPE_LOOKUP_DEPLOY_METER_MEMORY,
    e.STRIPE_LOOKUP_DEPLOY_METER_EGRESS,
    e.STRIPE_LOOKUP_DEPLOY_METER_DISK,
    e.STRIPE_LOOKUP_DEPLOY_METER_ACTIVE_KEYS,
  ];

  // All-or-nothing: a partially configured set would attach an incomplete
  // subscription (e.g. plan-fee with no metering), so treat it as unconfigured.
  if (!deployBillingConfigured()) {
    return null;
  }
  const planFee = planFeeLookupKeys as Record<DeployPlan, string>;
  const meters = meterLookupKeys as string[];

  const allLookupKeys = [...Object.values(planFee), ...meters];
  const cacheKey = allLookupKeys.join(",");
  const now = Date.now();
  if (cache && cache.key === cacheKey && cache.expiresAt > now) {
    return cache.config;
  }

  // One list call: <=10 lookup_keys, one active price each, so no paging.
  const stripe = getStripeClient();
  const prices = await stripe.prices.list({ lookup_keys: allLookupKeys, active: true, limit: 100 });
  const idByLookupKey = new Map<string, string>();
  for (const price of prices.data) {
    if (price.lookup_key) {
      idByLookupKey.set(price.lookup_key, price.id);
    }
  }

  // Every declared lookup_key must resolve to an active price, else the set is
  // incomplete: treat it as unconfigured rather than attach a partial sub.
  const planFeeEntries: Array<[DeployPlan, string]> = [];
  for (const plan of DEPLOY_PLANS) {
    const id = idByLookupKey.get(planFee[plan]);
    if (!id) {
      return null;
    }
    planFeeEntries.push([plan, id]);
  }
  const meteredPriceIds: string[] = [];
  for (const key of meters) {
    const id = idByLookupKey.get(key);
    if (!id) {
      return null;
    }
    meteredPriceIds.push(id);
  }
  const planFeePriceIds = Object.fromEntries(planFeeEntries) as Record<DeployPlan, string>;

  const config: DeployBillingConfig = {
    planFeePriceIds,
    meteredPriceIds,
    allDeployPriceIds: new Set<string>([...Object.values(planFeePriceIds), ...meteredPriceIds]),
  };
  cache = { key: cacheKey, expiresAt: now + RESOLUTION_TTL_MS, config };
  return config;
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
  return items.flatMap((item) => {
    const priceId = item.price?.id;
    return priceId && config.allDeployPriceIds.has(priceId) ? [{ id: item.id, priceId }] : [];
  });
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
