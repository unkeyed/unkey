import type { FixedSubscription, TieredSubscription } from "./subscriptions";
import { calculateTieredPrices } from "./tiers";

export function forecastUsage(currentUsage: number): number {
  const t = new Date();
  t.setUTCDate(1);
  t.setUTCHours(0, 0, 0, 0);

  const start = t.getTime();
  t.setUTCMonth(t.getUTCMonth() + 1);
  const end = t.getTime() - 1;

  const passed = (Date.now() - start) / (end - start);

  return currentUsage * (1 + 1 / passed);
}

type WorkspaceSubscriptions = {
  plan?: FixedSubscription;
  support?: FixedSubscription;
  activeKeys?: TieredSubscription;
  verifications?: TieredSubscription;
  ratelimits?: TieredSubscription;
};
type WorkspaceUsages = {
  activeKeys: number;
  verifications: number;
  ratelimits: number;
};

export function getCurrentUsageBill(
  usage: WorkspaceUsages,
  workspaceSubscriptions?: WorkspaceSubscriptions | null,
) {
  let currentPrice = 0;
  let estimatedTotalPrice = 0;

  if (workspaceSubscriptions?.plan) {
    const cost = Number.parseFloat(workspaceSubscriptions.plan.cents);
    currentPrice += cost;
    estimatedTotalPrice += cost; // does not scale
  }

  if (workspaceSubscriptions?.support) {
    const cost = Number.parseFloat(workspaceSubscriptions.support.cents);
    currentPrice += cost;
    estimatedTotalPrice += cost; // does not scale
  }

  if (workspaceSubscriptions?.activeKeys) {
    const cost = calculateTieredPrices(workspaceSubscriptions.activeKeys.tiers, usage.activeKeys);
    if (cost.err) {
      return {
        error: cost.err.message,
        currentPrice: 0,
        estimatedTotalPrice: 0,
      };
    }
    currentPrice += cost.val.totalCentsEstimate;
  }

  if (workspaceSubscriptions?.verifications) {
    const cost = calculateTieredPrices(
      workspaceSubscriptions.verifications.tiers,
      usage.verifications,
    );
    if (cost.err) {
      return {
        error: cost.err.message,
        currentPrice: 0,
        estimatedTotalPrice: 0,
      };
    }
    currentPrice += cost.val.totalCentsEstimate;
    estimatedTotalPrice += forecastUsage(cost.val.totalCentsEstimate);
  }

  if (workspaceSubscriptions?.ratelimits) {
    const cost = calculateTieredPrices(workspaceSubscriptions.ratelimits.tiers, usage.ratelimits);
    if (cost.err) {
      return {
        error: cost.err.message,
        currentPrice: 0,
        estimatedTotalPrice: 0,
      };
    }
    currentPrice += cost.val.totalCentsEstimate;
  }

  return { currentPrice, estimatedTotalPrice, error: null };
}
