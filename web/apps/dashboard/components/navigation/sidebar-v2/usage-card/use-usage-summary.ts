"use client";

import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import type { Route } from "next";

export type Measured<T> = { state: "loading" } | { state: "error" } | { state: "ready"; value: T };

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
  const { workspace, limits } = useWorkspace();
  const hasComputePlan = Boolean(workspace?.deployPlan) || Boolean(workspace?.deployPlanOverride);

  const deployUsage = trpc.billing.queryDeployUsage.useQuery(undefined, {
    enabled: hasComputePlan,
    staleTime: STALE_MS,
    refetchOnWindowFocus: false,
    trpc: { context: { skipBatch: true } },
  });
  const apiUsage = trpc.billing.queryUsage.useQuery(undefined, {
    staleTime: STALE_MS,
    refetchOnWindowFocus: false,
    trpc: { context: { skipBatch: true } },
  });

  if (!workspace) {
    return null;
  }

  const compute = hasComputePlan
    ? measureCompute(deployUsage, workspace.deploySpendBudgetCents ?? null)
    : null;
  const api = measureApi(apiUsage, limits?.apiBillableOperationsCountMaxPerMonth ?? null);

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
    return { state: "loading" };
  }
  return { state: "ready", value: { grossCents: usage.data.grossCents, budgetCents } };
}

function measureApi(
  usage: Query<{ billableTotal: number }>,
  max: number | null,
): Measured<ApiUsage> | null {
  if (max === null || max <= 0) {
    return null;
  }
  if (usage.isError) {
    return { state: "error" };
  }
  if (usage.data === undefined) {
    return { state: "loading" };
  }
  return { state: "ready", value: { used: usage.data.billableTotal, max } };
}
