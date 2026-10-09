"use client";

import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { policyListLoad } from "@/lib/collections/deploy/policies";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { useAppId, useProjectData } from "../../data-provider";
import { type MergedPolicy, type PolicyEnvs, mergePolicies } from "../components/list/merge";

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
  /** Writes replace whole lists, so they need both lists loaded. */
  canWrite: boolean;
};

export function usePoliciesData(): PoliciesData {
  const { environments, projectId, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();

  const productionEnv = environments.find((e) => e.kind === ENVIRONMENT_KIND.production);
  const previewEnv = environments.find((e) => e.kind === ENVIRONMENT_KIND.preview);

  const productionId = productionEnv?.id ?? "";
  const previewId = previewEnv?.id ?? "";

  const production = usePolicyRows(projectId, appId, productionId);
  const preview = usePolicyRows(projectId, appId, previewId);
  const productionRows = production.data;
  const previewRows = preview.data;

  const merged = useMemo(
    () => mergePolicies(productionRows ?? [], previewRows ?? []),
    [productionRows, previewRows],
  );

  const { failed: listFailed, retry } = useCollectionLoad(
    ...[productionId, previewId].filter((id) => id !== "").map(policyListLoad),
  );
  const isLoading = isEnvironmentsLoading || production.isLoading || preview.isLoading;
  const isError = production.isError || preview.isError || listFailed;

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
    retry,
    canWrite: !isLoading && !isError && productionId !== "",
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

/** One environment's rows in evaluation order. An empty id waits for the environment to load. */
function usePolicyRows(projectId: string, appId: string, environmentId: string) {
  return useLiveQuery(
    (q) =>
      environmentId
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
