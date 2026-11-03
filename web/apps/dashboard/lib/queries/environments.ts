"use client";
import { listAppEnvironments } from "@/lib/collections/deploy/app-environments";
import { queryKeys } from "@/lib/query-keys";
import { useQueries } from "@tanstack/react-query";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import type { environments as environmentsTable } from "@unkey/db/src/schema";
import { z } from "zod";

const kind = z.enum(["production", "preview"] as const satisfies readonly KindColumn[]);

/** Named access to the column values, e.g. `ENVIRONMENT_KIND.production`. */
export const ENVIRONMENT_KIND = kind.enum;
export const ENVIRONMENT_KINDS = kind.options;
export type EnvironmentKind = z.infer<typeof kind>;

export type Environment = {
  id: string;
  projectId: string;
  slug: string;
  kind: EnvironmentKind;
  appId: string;
};

export function productionFirst<T extends Pick<Environment, "kind">>(
  environments: readonly T[],
): T[] {
  return [...environments].sort(
    (a, b) => ENVIRONMENT_KINDS.indexOf(a.kind) - ENVIRONMENT_KINDS.indexOf(b.kind),
  );
}

// One query per app, as the API lists environments, merged once every app has loaded
export function useAppsEnvironments(apps: ReadonlyArray<{ projectId: string; appId: string }>) {
  const queries = useQueries({
    queries: apps.map(({ projectId, appId }) => appEnvironmentsOptions(projectId, appId)),
  });
  const isError = queries.some((query) => query.isError);
  const loaded = queries.every((query) => query.data !== undefined);
  return {
    data: loaded ? queries.flatMap((query) => query.data ?? []) : undefined,
    isLoading: queries.some((query) => query.isInitialLoading),
    isFetching: queries.some((query) => query.isFetching),
    isError,
    error: queries.find((query) => query.isError)?.error ?? null,
    refetch: () => Promise.all(queries.map((query) => query.refetch())),
  };
}

// A deleted app or one without read permission lists none, so it does not hide the other apps' environments
export async function loadAppEnvironments(
  projectId: string,
  appId: string,
): Promise<Environment[]> {
  try {
    const data = await listAppEnvironments(projectId, appId);
    return data.map((e) => ({ id: e.id, projectId, appId, slug: e.slug, kind: e.kind }));
  } catch (error) {
    if (error instanceof NotFoundErrorResponse) {
      return [];
    }
    throw error;
  }
}

export function appEnvironmentsOptions(projectId: string, appId: string) {
  return {
    queryKey: queryKeys.environments.app(projectId, appId),
    // The dashboard never writes environment rows
    staleTime: 5 * 60_000,
    retry: 3,
    queryFn: () => loadAppEnvironments(projectId, appId),
  };
}

/**
 * `environments.kind` as the database declares it. The table is imported as a
 * type only, so `satisfies` above breaks the build when the column drops a
 * value, and drizzle stays out of the browser bundle.
 */
type KindColumn = (typeof environmentsTable)["kind"]["enumValues"][number];
