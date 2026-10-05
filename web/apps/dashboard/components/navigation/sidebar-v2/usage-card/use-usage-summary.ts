"use client";

import { useWorkspaceLimits } from "@/hooks/use-workspace-limits";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import type { Route } from "next";

/** `bar` says whether the loaded row will draw a bar, so the loading row matches it */
export type Measured<T> =
  | { state: "loading"; bar: boolean }
  | { state: "error" }
  | { state: "ready"; value: T };

export type ComputeUsage = {
  grossCents: number;
  budgetCents: number | null;
};

export type ApiUsage = {
  used: number;
  max: number;
};

export type UsageSummary = {
  href: Route;
  compute: Measured<ComputeUsage> | null;
  api: Measured<ApiUsage> | null;
  atRisk: boolean;
};

export const AT_RISK = 0.9;

const STALE_MS = 5 * 60 * 1000;

export function useUsageSummary(): UsageSummary | null {
  const { workspace } = useWorkspace();
  const hasComputePlan = Boolean(workspace?.deployPlan) || Boolean(workspace?.deployPlanOverride);

  const deployUsage = trpc.billing.queryDeployUsage.useQuery(undefined, {
    enabled: hasComputePlan,
    staleTime: STALE_MS,
    refetchOnWindowFocus: false,
    trpc: { context: { skipBatch: true } },
  });
  const apiUsage = useWorkspaceLimits({
    staleTime: STALE_MS,
    refetchOnWindowFocus: false,
  });

  if (!workspace) {
    return null;
  }

  const compute = hasComputePlan
    ? measureCompute(deployUsage, workspace.deploySpendBudgetCents ?? null)
    : null;
  const api = measureApi(apiUsage);

  return {
    href: routes.settings.usage({ workspaceSlug: workspace.slug }),
    compute,
    api,
    atRisk: overCeiling(compute, computeRatio) || overCeiling(api, apiRatio),
  };
}

export function computeRatio(value: ComputeUsage): number | null {
  if (value.budgetCents === null || value.budgetCents <= 0) {
    return null;
  }
  return value.grossCents / value.budgetCents;
}

export function apiRatio(value: ApiUsage): number {
  return value.used / value.max;
}

function overCeiling<T>(
  measured: Measured<T> | null,
  ratioOf: (value: T) => number | null,
): boolean {
  if (measured === null || measured.state !== "ready") {
    return false;
  }
  const ratio = ratioOf(measured.value);
  return ratio !== null && ratio >= AT_RISK;
}

type Query<T> = { data: T | undefined; isError: boolean };

function measureCompute(
  usage: Query<{ grossCents: number }>,
  budgetCents: number | null,
): Measured<ComputeUsage> {
  if (usage.isError) {
    return { state: "error" };
  }
  if (usage.data === undefined) {
    return { state: "loading", bar: budgetCents !== null && budgetCents > 0 };
  }
  return { state: "ready", value: { grossCents: usage.data.grossCents, budgetCents } };
}

function measureApi(
  usage: Query<{ api: { billableOperations: { used: number; limit: number } } }>,
): Measured<ApiUsage> | null {
  if (usage.isError) {
    return { state: "error" };
  }
  if (usage.data === undefined) {
    return { state: "loading", bar: true };
  }
  const { used, limit } = usage.data.api.billableOperations;
  if (limit <= 0) {
    return null;
  }
  return { state: "ready", value: { used, max: limit } };
}
