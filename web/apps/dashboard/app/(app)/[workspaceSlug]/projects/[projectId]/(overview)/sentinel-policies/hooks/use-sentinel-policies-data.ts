"use client";

import { collection } from "@/lib/collections";
import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { useProjectData } from "../../data-provider";
import { type MergedPolicy, mergePolicies } from "../components/list/merge";

type SentinelPoliciesData = {
  envAId: string;
  envBId: string;
  envASlug: string;
  envBSlug: string;
  merged: MergedPolicy[];
  isLoading: boolean;
  isError: boolean;
};

/**
 * Loads the two env-scoped sentinel policy lists, strips row-only fields,
 * and merges them into the row shape consumed by `SentinelPoliciesList`.
 */
export function useSentinelPoliciesData(): SentinelPoliciesData {
  const { environments } = useProjectData();

  const envA = environments.find((e) => e.slug === "production") ?? environments.at(0);
  const envB = environments.find((e) => e.id !== envA?.id) ?? environments.at(1);

  const envAId = envA?.id ?? "";
  const envBId = envB?.id ?? "";
  const envASlug = envA?.slug ?? "production";
  const envBSlug = envB?.slug ?? "preview";

  const {
    data: rowsA,
    isLoading: isLoadingA,
    isError: isErrorA,
  } = useLiveQuery(
    (q) =>
      q
        .from({ p: collection.sentinelPolicies })
        .where(({ p }) => eq(p.environmentId, envAId))
        .orderBy(({ p }) => p._order),
    [envAId],
  );

  const {
    data: rowsB,
    isLoading: isLoadingB,
    isError: isErrorB,
  } = useLiveQuery(
    (q) =>
      q
        .from({ p: collection.sentinelPolicies })
        .where(({ p }) => eq(p.environmentId, envBId))
        .orderBy(({ p }) => p._order),
    [envBId],
  );

  const merged = useMemo(() => {
    const policiesA: SentinelPolicy[] = rowsA.map(
      ({ environmentId: _e, _order: _o, ...p }) => p as SentinelPolicy,
    );
    const policiesB: SentinelPolicy[] = rowsB.map(
      ({ environmentId: _e, _order: _o, ...p }) => p as SentinelPolicy,
    );
    return mergePolicies(policiesA, policiesB);
  }, [rowsA, rowsB]);

  return {
    envAId,
    envBId,
    envASlug,
    envBSlug,
    merged,
    isLoading: isLoadingA || isLoadingB,
    isError: isErrorA || isErrorB,
  };
}
