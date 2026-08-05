import { type PlanLimits, limitsByPlan } from "@/lib/quotas";
import type { Quotas } from "@unkey/db";
import type Stripe from "stripe";

/**
 * The Unkey Deploy plans we recognize, lowest to highest. Mirrored into
 * workspaces.deploy_plan; NULL (absence) means no Deploy plan.
 *
 * Adding a plan: create its Stripe price(s) tagged with metadata `plan=<name>`
 * and add the name here. No price ids are hardcoded, so re-pricing an existing
 * plan needs no change at all: the new price carries the same metadata and
 * existing subscriptions keep resolving (grandfathered).
 */
export const DEPLOY_PLANS = ["starter", "pro", "business"] as const;
export type DeployPlan = (typeof DEPLOY_PLANS)[number];

type ComputeQuotas = Pick<
  Quotas,
  | "logsRetentionDays"
  | "auditLogsRetentionDays"
  | "team"
  | "maxCpuMillicoresPerInstance"
  | "maxMemoryMibPerInstance"
  | "maxStorageMibPerInstance"
  | "maxConcurrentBuilds"
>;

type ComputeOnlyQuotas = Pick<
  ComputeQuotas,
  | "maxCpuMillicoresPerInstance"
  | "maxMemoryMibPerInstance"
  | "maxStorageMibPerInstance"
  | "maxConcurrentBuilds"
>;

function computeQuotasFromLimits(limits: PlanLimits): ComputeQuotas {
  return {
    logsRetentionDays: limits.logsRetentionDaysMax,
    auditLogsRetentionDays: limits.logsAuditRetentionDaysMax,
    team: limits.teamEnabled,
    maxCpuMillicoresPerInstance: limits.cpuCoresMaxPerInstance * 1_000,
    maxMemoryMibPerInstance: limits.memoryMibMaxPerInstance,
    maxStorageMibPerInstance: limits.storageMibMaxPerInstance,
    maxConcurrentBuilds: limits.buildsConcurrentMax,
  };
}

export function computeQuotasForPlan(plan: DeployPlan | null): ComputeQuotas {
  return computeQuotasFromLimits(limitsByPlan[plan ?? "free"]);
}

/**
 * Returns the quota fields a Compute subscription may safely update. A paid
 * API plan owns the shared retention and team fields, so Compute must preserve
 * those while still applying its resource limits.
 */
export function computeQuotaUpdateForPlan(
  plan: DeployPlan | null,
  preserveApiQuotas: boolean,
): ComputeQuotas | ComputeOnlyQuotas {
  const quotas = computeQuotasForPlan(plan);
  if (!preserveApiQuotas) {
    return quotas;
  }
  return {
    maxCpuMillicoresPerInstance: quotas.maxCpuMillicoresPerInstance,
    maxMemoryMibPerInstance: quotas.maxMemoryMibPerInstance,
    maxStorageMibPerInstance: quotas.maxStorageMibPerInstance,
    maxConcurrentBuilds: quotas.maxConcurrentBuilds,
  };
}

export function deployPlanGrantsTeam(plan: string | null): boolean {
  return plan === "pro" || plan === "business";
}

export function parseDeployPlan(plan: string | null): DeployPlan | null {
  return plan !== null && isDeployPlan(plan) ? plan : null;
}

function isDeployPlan(value: string): value is DeployPlan {
  return (DEPLOY_PLANS as readonly string[]).includes(value);
}

/**
 * Detects which Deploy plan, if any, a Stripe subscription carries.
 *
 * The plan-fee price is tagged in Stripe with metadata `plan=<plan>`, the
 * canonical "has a plan" marker. We scan the subscription's items (the plan-fee
 * sits alongside API items and metered Deploy prices) and return the first
 * recognized plan. price.metadata ships in the webhook payload, so this needs
 * no extra Stripe calls.
 *
 * Fails closed: an item tagged with an unrecognized plan is logged and ignored,
 * so a Stripe typo can never grant Deploy access. Returns null when no item
 * carries a recognized Deploy plan.
 */
export function detectDeployPlan(sub: Stripe.Subscription): DeployPlan | null {
  for (const item of sub.items?.data ?? []) {
    const raw = item.price?.metadata?.plan?.trim();
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
