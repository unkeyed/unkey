"use client";

import { collection } from "@/lib/collections";
import { queryClient } from "@/lib/collections/client";
import { type App, refetchProjectApps } from "@/lib/collections/deploy/apps";
import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import type { Environment } from "@/lib/collections/deploy/environments";
import { pickPrimaryApp } from "@/lib/collections/deploy/project-cards";
import { queryKeys } from "@/lib/query-keys";
import {
  type InitialQueryBuilder,
  and,
  createLiveQueryCollection,
  eq,
  inArray,
  useLiveQuery,
} from "@tanstack/react-db";
import { useCallback, useSyncExternalStore } from "react";
import { useAppScope, useDeployScope } from "./context";

// Shaped like a React Query result so the source behind a hook can change
// without touching its callers. Lists are empty, not undefined, while loading.
const MEMO_ENTRIES_MAX = 50;

export type QueryResult<T> = {
  data: T;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  refetch: () => Promise<unknown>;
};

export function useProjectApps(): QueryResult<App[]> {
  return useAppsInProject(useDeployScope().projectId);
}

// A project's apps, latest update first. Needs no DeployDataProvider.
export function useAppsInProject(
  projectId: string,
  { enabled = true }: { enabled?: boolean } = {},
): QueryResult<App[]> {
  const query = useLiveQuery(
    () => (enabled ? projectAppsQuery(projectId) : null),
    [projectId, enabled],
  );
  const load = useLoadState(collection.apps.utils, refetchProjectApps, projectId);
  return { data: query.data ?? [], isLoading: query.isLoading, ...load };
}

// The scope's app, or the project's primary app outside an app
export function useApp(): QueryResult<App | undefined> {
  const { appId } = useDeployScope();
  const apps = useProjectApps();
  return {
    ...apps,
    data: appId ? apps.data.find((app) => app.id === appId) : pickPrimaryApp(apps.data),
  };
}

// The app's environments, or those of every app in the project outside an app
export function useEnvironments(): QueryResult<Environment[]> {
  const { projectId, appId } = useDeployScope();
  const apps = useProjectApps();
  const appIds = appId
    ? appId
    : apps.data
        .map((app) => app.id)
        .sort()
        .join(",");
  const query = useLiveQuery(
    () => (appIds === "" ? null : environmentsQuery(projectId, appIds)),
    [projectId, appIds],
  );
  const load = useLoadState(collection.environments.utils, refetchScopeEnvironments, projectId);
  return {
    data: query.data ?? [],
    isLoading: (appId === undefined && apps.isLoading) || query.isLoading,
    ...load,
  };
}

export function useDomains(): QueryResult<Domain[]> {
  const { projectId, appId } = useDeployScope();
  const query = useLiveQuery(() => domainsQuery(projectId, appId ?? null), [projectId, appId]);
  const load = useLoadState(collection.domains.utils, refetchScopeDomains, projectId);
  return { data: query.data ?? [], isLoading: query.isLoading, ...load };
}

export function useCustomDomains(): QueryResult<CustomDomain[]> {
  const { projectId, appId } = useDeployScope();
  const query = useLiveQuery(
    () => customDomainsQuery(projectId, appId ?? null),
    [projectId, appId],
  );
  const load = useLoadState(collection.customDomains.utils, refetchScopeCustomDomains, projectId);
  return { data: query.data ?? [], isLoading: query.isLoading, ...load };
}

export function useEnvVars(projectId: string, appId: string): QueryResult<EnvVar[]> {
  const query = useLiveQuery(() => envVarsQuery(projectId, appId), [projectId, appId]);
  const load = useLoadState(collection.envVars.utils, refetchAppEnvVars, appId);
  return { data: query.data ?? [], isLoading: query.isLoading, ...load };
}

// The settings of every environment of the scope's app
export function useAppEnvironmentSettings(): QueryResult<EnvironmentSettings[]> {
  const { projectId, appId } = useAppScope();
  const query = useLiveQuery(() => environmentSettingsQuery(projectId, appId), [projectId, appId]);
  const load = useLoadState(
    collection.environmentSettings.utils,
    refetchAppEnvironmentSettings,
    appId,
  );
  return { data: query.data ?? [], isLoading: query.isLoading, ...load };
}

// Reads what the apps already loaded hold, without subscribing, for event handlers
export function getAppData(appId: string): App | undefined {
  return collection.apps.get(appId);
}

export function getProjectAppsData(projectId: string): App[] {
  return collection.apps.toArray.filter((app) => app.projectId === projectId);
}

// Drops an app the server no longer has, without waiting for a refetch
export function removeAppData(appId: string): void {
  if (collection.apps.has(appId)) {
    collection.apps.utils.writeDelete(appId);
  }
}

export function refetchAllApps() {
  return collection.apps.utils.refetch();
}

export function refetchScopeDeployments(projectId: string, appId: string | undefined) {
  return queryClient.refetchQueries({
    queryKey: queryKeys.deployments.scope(projectId, appId),
    type: "active",
  });
}

export function refetchScopeDomains(projectId: string) {
  return queryClient.refetchQueries({ queryKey: ["domains", projectId], type: "active" });
}

export function refetchScopeCustomDomains(projectId: string) {
  return queryClient.refetchQueries({ queryKey: ["customDomains", projectId], type: "active" });
}

// Every read of the scope, after an awaited change settles
export function refetchDeployData(projectId: string, appId: string | undefined) {
  return Promise.all([
    refetchProjectApps(projectId),
    refetchScopeDeployments(projectId, appId),
    refetchScopeDomains(projectId),
    refetchScopeCustomDomains(projectId),
    refetchScopeEnvironments(projectId),
  ]);
}

/**
 * Each hook call site gets the same live query for the same arguments, so
 * components reading one thing share a single subscription instead of building
 * one each. An unused live query cleans itself up and restarts on next use.
 * Hovering and opening deployments adds one query per id, so only the most
 * recently used ones are kept; a component holding an evicted one keeps it.
 */
export function memoByArgs<A extends Array<string | number | null>, T>(
  create: (...args: A) => T,
): (...args: A) => T {
  const cache = new Map<string, T>();
  return (...args) => {
    const key = JSON.stringify(args);
    const cached = cache.get(key);
    if (cached !== undefined) {
      cache.delete(key);
      cache.set(key, cached);
      return cached;
    }
    const created = create(...args);
    cache.set(key, created);
    if (cache.size > MEMO_ENTRIES_MAX) {
      const [oldest] = cache.keys();
      if (oldest !== undefined) {
        cache.delete(oldest);
      }
    }
    return created;
  };
}

/**
 * A live query reports only errors inside sync, so load failures are read from
 * the source collection. A failed load records its error there and emits
 * nothing, so the flag is reread on every query cache event.
 */
export function useLoadState(
  source: LoadSource,
  refetchScope: (scopeId: string) => Promise<unknown>,
  scopeId: string,
): Pick<QueryResult<unknown>, "isError" | "error" | "refetch"> {
  const isError = useSyncExternalStore(
    subscribeQueryCache,
    () => source.isError,
    () => false,
  );
  const refetch = useCallback(
    () => (source.isError ? source.clearError() : refetchScope(scopeId)),
    [source, refetchScope, scopeId],
  );
  return { isError, error: isError ? source.lastError : null, refetch };
}

export function subscribeQueryCache(onChange: () => void): () => void {
  return queryClient.getQueryCache().subscribe(onChange);
}

export function projectAppsQueryFor(projectId: string) {
  return (q: InitialQueryBuilder) =>
    q
      .from({ app: collection.apps })
      .where(({ app }) => eq(app.projectId, projectId))
      .orderBy(({ app }) => app.updatedAt, { direction: "desc", nulls: "last" })
      .orderBy(({ app }) => app.id, "desc");
}

export function domainsQueryFor(projectId: string, appId: string | null) {
  return (q: InitialQueryBuilder) =>
    q
      .from({ domain: collection.domains })
      .where(({ domain }) =>
        appId
          ? and(eq(domain.projectId, projectId), eq(domain.appId, appId))
          : eq(domain.projectId, projectId),
      )
      .orderBy(({ domain }) => domain.createdAt, "desc");
}

export function customDomainsQueryFor(projectId: string, appId: string | null) {
  return (q: InitialQueryBuilder) =>
    q
      .from({ customDomain: collection.customDomains })
      .where(({ customDomain }) =>
        appId
          ? and(eq(customDomain.projectId, projectId), eq(customDomain.appId, appId))
          : eq(customDomain.projectId, projectId),
      )
      .orderBy(({ customDomain }) => customDomain.createdAt, "desc");
}

export function environmentsQueryFor(projectId: string, appIds: string) {
  return (q: InitialQueryBuilder) =>
    q
      .from({ env: collection.environments })
      .where(({ env }) => and(eq(env.projectId, projectId), inArray(env.appId, appIds.split(","))));
}

const projectAppsQuery = memoByArgs((projectId: string) =>
  createLiveQueryCollection(projectAppsQueryFor(projectId)),
);

const envVarsQuery = memoByArgs((projectId: string, appId: string) =>
  createLiveQueryCollection((q) =>
    q
      .from({ v: collection.envVars })
      .where(({ v }) => and(eq(v.projectId, projectId), eq(v.appId, appId))),
  ),
);

const environmentSettingsQuery = memoByArgs((projectId: string, appId: string) =>
  createLiveQueryCollection((q) =>
    q
      .from({ s: collection.environmentSettings })
      .where(({ s }) => and(eq(s.projectId, projectId), eq(s.appId, appId))),
  ),
);

const environmentsQuery = memoByArgs((projectId: string, appIds: string) =>
  createLiveQueryCollection(environmentsQueryFor(projectId, appIds)),
);

const domainsQuery = memoByArgs((projectId: string, appId: string | null) =>
  createLiveQueryCollection(domainsQueryFor(projectId, appId)),
);

const customDomainsQuery = memoByArgs((projectId: string, appId: string | null) =>
  createLiveQueryCollection(customDomainsQueryFor(projectId, appId)),
);

function refetchAppEnvVars(appId: string) {
  return queryClient.refetchQueries({ queryKey: ["envVars", appId], type: "active" });
}

function refetchAppEnvironmentSettings(appId: string) {
  return queryClient.refetchQueries({ queryKey: ["environmentSettings", appId], type: "active" });
}

// Environment keys name each app as projectId:appId
function refetchScopeEnvironments(projectId: string) {
  return queryClient.refetchQueries({
    type: "active",
    predicate: ({ queryKey }) =>
      queryKey[0] === "environments" &&
      queryKey.some((part) => typeof part === "string" && part.startsWith(`${projectId}:`)),
  });
}

type LoadSource = {
  readonly isError: boolean;
  readonly lastError: unknown;
  clearError: () => Promise<void>;
};
