"use client";

import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { type PolicyRow, policyListLoad } from "@/lib/collections/deploy/policies";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { useAppId, useProjectData } from "../../data-provider";
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
  const { environments, projectId, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();

  const productionEnv = environments.find((e) => e.kind === ENVIRONMENT_KIND.production);
  const previewEnv = environments.find((e) => e.kind === ENVIRONMENT_KIND.preview);

  const productionId = productionEnv?.id ?? null;
  const previewId = previewEnv?.id ?? null;

  const production = useEnvPolicies(projectId, appId, productionId);
  const preview = useEnvPolicies(projectId, appId, previewId);
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
};

export function useProductionPolicies(): ProductionPolicies {
  const { environments, projectId, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();
  const productionId = environments.find((e) => e.kind === ENVIRONMENT_KIND.production)?.id ?? null;
  const production = useEnvPolicies(projectId, appId, productionId);
  const policies = production.rows ?? [];

  return {
    policies,
    view: policiesView(
      production.isError,
      isEnvironmentsLoading || production.isLoading,
      policies.length,
    ),
    retry: production.retry,
  };
}

function useEnvPolicies(projectId: string, appId: string, environmentId: string | null) {
  const rows = usePolicyRows(projectId, appId, environmentId);
  const { failed, retry } = useCollectionLoad(
    ...(environmentId === null ? [] : [policyListLoad(environmentId)]),
  );
  return {
    rows: rows.data,
    isLoading: rows.isLoading,
    isError: rows.isError || failed,
    retry,
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

function usePolicyRows(projectId: string, appId: string, environmentId: string | null) {
  return useLiveQuery(
    (q) =>
      environmentId !== null
        ? q
            .from({ p: collection.policies })
            .where(({ p }) =>
              and(
                eq(p.projectId, projectId),
                eq(p.appId, appId),
                eq(p.environmentId, environmentId),
              ),
            )
            .orderBy(({ p }) => p._order)
        : null,
    [projectId, appId, environmentId],
  );
}
