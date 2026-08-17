"use client";

import { collection } from "@/lib/collections";
import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { useProjectData } from "../../data-provider";
import { type MergedPolicy, mergePolicies } from "../components/list/merge";

type PoliciesData = {
  envAId: string;
  envBId: string;
  envASlug: string;
  envBSlug: string;
  merged: MergedPolicy[];
  isLoading: boolean;
  isError: boolean;
};

/**
 * Loads the two environment-scoped policy lists, strips row-only fields,
 * and merges them into the row shape consumed by `PoliciesList`.
 */
export function usePoliciesData(): PoliciesData {
  const { environments } = useProjectData();

  const envA = environments.find((e) => e.kind === "production") ?? environments.at(0);
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
        .from({ p: collection.policies })
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
        .from({ p: collection.policies })
        .where(({ p }) => eq(p.environmentId, envBId))
        .orderBy(({ p }) => p._order),
    [envBId],
  );

  const merged = useMemo(() => {
    const policiesA: Policy[] = rowsA.map(({ environmentId: _e, _order: _o, ...p }) => p as Policy);
    const policiesB: Policy[] = rowsB.map(({ environmentId: _e, _order: _o, ...p }) => p as Policy);
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
