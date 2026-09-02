"use client";

import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import type { PolicyRow } from "@/lib/collections/deploy/policies";
import { useAppId, useProjectData } from "../../data-provider";
import { type Env, type MergedPolicy, mergePolicies } from "../components/list/merge";

type PoliciesData = {
  productionId: string;
  previewId: string;
  productionSlug: string;
  previewSlug: string;
  merged: MergedPolicy[];
  /**
   * Each environment's rows in its own evaluation order. Writes need this, not
   * `merged`: `merged` follows production, so building preview's list from it
   * would reorder preview.
   */
  rowsByEnv: Record<Env, PolicyRow[]>;
  isLoading: boolean;
  isError: boolean;
};

export function usePoliciesData(): PoliciesData {
  const { environments, projectId } = useProjectData();
  const appId = useAppId();

  const production = environments.find((e) => e.kind === ENVIRONMENT_KIND.production);
  const preview = environments.find((e) => e.kind === ENVIRONMENT_KIND.preview);

  const productionId = production?.id ?? "";
  const previewId = preview?.id ?? "";
  const productionSlug = production?.slug ?? ENVIRONMENT_KIND.production;
  const previewSlug = preview?.slug ?? ENVIRONMENT_KIND.preview;

  const {
    data: productionRows,
    isLoading: isLoadingProduction,
    isError: isErrorProduction,
  } = useLiveQuery(
    (q) =>
      q
        .from({ p: collection.policies })
        .where(({ p }) =>
          and(eq(p.projectId, projectId), eq(p.appId, appId), eq(p.environmentId, productionId)),
        )
        .orderBy(({ p }) => p._order),
    [projectId, appId, productionId],
  );

  const {
    data: previewRows,
    isLoading: isLoadingPreview,
    isError: isErrorPreview,
  } = useLiveQuery(
    (q) =>
      q
        .from({ p: collection.policies })
        .where(({ p }) =>
          and(eq(p.projectId, projectId), eq(p.appId, appId), eq(p.environmentId, previewId)),
        )
        .orderBy(({ p }) => p._order),
    [projectId, appId, previewId],
  );

  const merged = useMemo(
    () => mergePolicies(productionRows, previewRows),
    [productionRows, previewRows],
  );
  const rowsByEnv = useMemo(
    () => ({ production: productionRows, preview: previewRows }),
    [productionRows, previewRows],
  );

  return {
    productionId,
    previewId,
    productionSlug,
    previewSlug,
    merged,
    rowsByEnv,
    isLoading: isLoadingProduction || isLoadingPreview,
    isError: isErrorProduction || isErrorPreview,
  };
}
