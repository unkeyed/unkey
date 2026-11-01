"use client";

import { ENVIRONMENT_KIND, type PolicyRow, useEnvironments, usePolicies } from "@/lib/deploy-data";
import { useMemo, useState } from "react";
import { type MergedPolicy, type PolicyEnvs, mergePolicies } from "../model/merge";

export type PoliciesView =
  | { type: "error" }
  | { type: "loading" }
  | { type: "empty" }
  | { type: "list" };

export type PoliciesData = {
  envs: PolicyEnvs;
  merged: MergedPolicy[];
  view: PoliciesView;
  retry: () => void;
  canWrite: boolean;
};

export function usePoliciesData(): PoliciesData {
  const { data: environments, isLoading: isEnvironmentsLoading } = useEnvironments();

  const productionEnv = environments.find((e) => e.kind === ENVIRONMENT_KIND.production);
  const previewEnv = environments.find((e) => e.kind === ENVIRONMENT_KIND.preview);

  const productionId = productionEnv?.id ?? null;
  const previewId = previewEnv?.id ?? null;

  const production = useEnvPolicies(productionId);
  const preview = useEnvPolicies(previewId);
  const productionRows = production.rows;
  const previewRows = preview.rows;

  const merged = useMemo(
    () => mergePolicies(productionRows ?? [], previewRows ?? []),
    [productionRows, previewRows],
  );

  const isLoading = isEnvironmentsLoading || production.isLoading || preview.isLoading;
  const isError = production.isError || preview.isError;

  const envs = useMemo(
    () => ({
      production: { id: productionId, slug: productionEnv?.slug ?? ENVIRONMENT_KIND.production },
      preview: { id: previewId, slug: previewEnv?.slug ?? ENVIRONMENT_KIND.preview },
    }),
    [productionId, previewId, productionEnv?.slug, previewEnv?.slug],
  );

  return {
    envs,
    merged,
    view: policiesView(isError, isLoading, merged.length),
    retry: () => {
      production.retry();
      preview.retry();
    },
    canWrite: !isLoading && !isError && productionId !== null && previewId !== null,
  };
}

export type ProductionPolicies = {
  policies: PolicyRow[];
  view: PoliciesView;
  retry: () => void;
  isRetrying: boolean;
};

export function useProductionPolicies(): ProductionPolicies {
  const { data: environments, isLoading: isEnvironmentsLoading } = useEnvironments();
  const productionId = environments.find((e) => e.kind === ENVIRONMENT_KIND.production)?.id ?? null;
  const production = useEnvPolicies(productionId);
  const policies = production.rows ?? [];

  return {
    policies,
    view: policiesView(
      production.isError,
      isEnvironmentsLoading || production.isLoading,
      policies.length,
    ),
    retry: production.retry,
    isRetrying: production.isRetrying,
  };
}

function useEnvPolicies(environmentId: string | null) {
  const policies = usePolicies(environmentId);
  const [retrying, setRetrying] = useState(false);
  const retry = () => {
    setRetrying(true);
    policies.refetch().finally(() => setRetrying(false));
  };
  return {
    rows: policies.data,
    isLoading: policies.isLoading,
    isError: policies.isError,
    retry,
    isRetrying: retrying,
  };
}

function policiesView(isError: boolean, isLoading: boolean, count: number): PoliciesView {
  if (isError) {
    return { type: "error" };
  }
  if (isLoading) {
    return { type: "loading" };
  }
  return count === 0 ? { type: "empty" } : { type: "list" };
}
