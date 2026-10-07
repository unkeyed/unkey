"use client";

import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { type PolicyRow, policyListLoad } from "@/lib/collections/deploy/policies";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { useAppId, useProjectData } from "../../data-provider";
import {
  type Env,
  type MergedPolicy,
  type PolicyEnvs,
  mergePolicies,
} from "../components/list/merge";

export type PoliciesData = {
  envs: PolicyEnvs;
  merged: MergedPolicy[];
  /**
   * Each environment's rows in its own evaluation order. Writes need this, not
   * `merged`: `merged` follows production, so building preview's list from it
   * would reorder preview.
   */
  rowsByEnv: Record<Env, PolicyRow[]>;
  isLoading: boolean;
  isError: boolean;
  /** Writes replace whole lists, so they need both lists loaded. */
  canWrite: boolean;
};

export function usePoliciesData(): PoliciesData {
  const { environments, projectId, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();

  const production = environments.find((e) => e.kind === ENVIRONMENT_KIND.production);
  const preview = environments.find((e) => e.kind === ENVIRONMENT_KIND.preview);

  const productionId = production?.id ?? "";
  const previewId = preview?.id ?? "";

  const {
    data: productionRows,
    isLoading: isLoadingProduction,
    isError: isErrorProduction,
  } = useLiveQuery(
    (q) =>
      productionId
        ? q
            .from({ p: collection.policies })
            .where(({ p }) =>
              and(
                eq(p.projectId, projectId),
                eq(p.appId, appId),
                eq(p.environmentId, productionId),
              ),
            )
            .orderBy(({ p }) => p._order)
        : null,
    [projectId, appId, productionId],
  );

  const {
    data: previewRows,
    isLoading: isLoadingPreview,
    isError: isErrorPreview,
  } = useLiveQuery(
    (q) =>
      previewId
        ? q
            .from({ p: collection.policies })
            .where(({ p }) =>
              and(eq(p.projectId, projectId), eq(p.appId, appId), eq(p.environmentId, previewId)),
            )
            .orderBy(({ p }) => p._order)
        : null,
    [projectId, appId, previewId],
  );

  const merged = useMemo(
    () => mergePolicies(productionRows ?? [], previewRows ?? []),
    [productionRows, previewRows],
  );
  const rowsByEnv = useMemo(
    () => ({ production: productionRows ?? [], preview: previewRows ?? [] }),
    [productionRows, previewRows],
  );

  const { failed: listFailed } = useCollectionLoad(
    ...[productionId, previewId].filter((id) => id !== "").map(policyListLoad),
  );
  const isLoading = isEnvironmentsLoading || isLoadingProduction || isLoadingPreview;
  const isError = isErrorProduction || isErrorPreview || listFailed;

  const envs = useMemo(
    () => ({
      production: { id: productionId, slug: production?.slug ?? ENVIRONMENT_KIND.production },
      preview: { id: previewId, slug: preview?.slug ?? ENVIRONMENT_KIND.preview },
    }),
    [productionId, previewId, production?.slug, preview?.slug],
  );

  return {
    envs,
    merged,
    rowsByEnv,
    isLoading,
    isError,
    canWrite: !isLoading && !isError && productionId !== "",
  };
}
