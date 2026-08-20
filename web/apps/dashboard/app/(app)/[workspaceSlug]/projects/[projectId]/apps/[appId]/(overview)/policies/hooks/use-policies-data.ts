"use client";

import { collection } from "@/lib/collections";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { useAppId, useProjectData } from "../../data-provider";
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

/** Loads both environments' policies in parallel and merges them into rows. */
export function usePoliciesData(): PoliciesData {
  const { environments, projectId } = useProjectData();
  const appId = useAppId();

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
        .where(({ p }) =>
          and(eq(p.projectId, projectId), eq(p.appId, appId), eq(p.environmentId, envAId)),
        )
        .orderBy(({ p }) => p._order),
    [projectId, appId, envAId],
  );

  const {
    data: rowsB,
    isLoading: isLoadingB,
    isError: isErrorB,
  } = useLiveQuery(
    (q) =>
      q
        .from({ p: collection.policies })
        .where(({ p }) =>
          and(eq(p.projectId, projectId), eq(p.appId, appId), eq(p.environmentId, envBId)),
        )
        .orderBy(({ p }) => p._order),
    [projectId, appId, envBId],
  );

  const merged = useMemo(() => mergePolicies(rowsA, rowsB), [rowsA, rowsB]);

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
