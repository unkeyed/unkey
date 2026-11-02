"use client";

import { collection } from "@/lib/collections";
import { queryClient } from "@/lib/collections/client";
import {
  type CustomDomain,
  retryDomainVerification,
} from "@/lib/collections/deploy/custom-domains";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment } from "@/lib/collections/deploy/environments";
import type { QueryResult } from "@/lib/loadable/loadable";
import { type App, refetchProjectApps, useProjectApps } from "@/lib/queries/apps";
import {
  type InitialQueryBuilder,
  and,
  createLiveQueryCollection,
  eq,
  inArray,
  useLiveQuery,
} from "@tanstack/react-db";
import type { QueryClient } from "@tanstack/react-query";
import { useCallback, useSyncExternalStore } from "react";
import { useDeployScope } from "./context";

// Data stays undefined until the first load, so fromQuery tells loading from empty
export type DeployQuery<T> = QueryResult<T> & { isLoading: boolean };

export function useScopeApps(): DeployQuery<App[]> {
  return useProjectApps(useDeployScope().projectId);
}

// The scope's app, or outside an app the most recently updated deployed one, since apps
// arrive newest first. Null when it does not exist.
export function useApp(): DeployQuery<App | null> {
  const { appId } = useDeployScope();
  const apps = useScopeApps();
  return {
    ...apps,
    data:
      apps.data === undefined
        ? undefined
        : (apps.data.find((app) => (appId ? app.id === appId : app.currentDeploymentId !== null)) ??
          null),
  };
}

// The app's environments, or those of every app in the project outside an app
export function useEnvironments(): DeployQuery<Environment[]> {
  const { projectId, appId } = useDeployScope();
  const apps = useScopeApps();
  const appIds = appId
    ? appId
    : (apps.data ?? [])
        .map((app) => app.id)
        .sort()
        .join(",");
  const query = useLiveQuery(
    () => (appIds === "" ? null : environmentsQuery(projectId, appIds)),
    [projectId, appIds],
  );
  const load = useLoadState(collection.environments.utils, refetchScopeEnvironments, projectId);
  return liveResult(
    { data: query.data, isLoading: (appId === undefined && apps.isLoading) || query.isLoading },
    appId === undefined && apps.isError ? apps : load,
  );
}

export function useDomains(): DeployQuery<Domain[]> {
  const { projectId, appId } = useDeployScope();
  const query = useLiveQuery(() => domainsQuery(projectId, appId ?? null), [projectId, appId]);
  const load = useLoadState(collection.domains.utils, refetchScopeDomains, projectId);
  return liveResult(query, load);
}

export function useCustomDomains(): DeployQuery<CustomDomain[]> {
  const { projectId, appId } = useDeployScope();
  const query = useLiveQuery(
    () => customDomainsQuery(projectId, appId ?? null),
    [projectId, appId],
  );
  const load = useLoadState(collection.customDomains.utils, refetchScopeCustomDomains, projectId);
  return liveResult(query, load);
}

export function useCustomDomainMutations() {
  return CUSTOM_DOMAIN_MUTATIONS;
}

export function refetchScopeDomains(projectId: string) {
  return queryClient.refetchQueries({ queryKey: ["domains", projectId], type: "active" });
}

export function refetchScopeCustomDomains(projectId: string) {
  return queryClient.refetchQueries({ queryKey: ["customDomains", projectId], type: "active" });
}

// Every read of the scope outside deployments, after an awaited change settles
export function refetchDeployData(reactQueryClient: QueryClient, projectId: string) {
  return Promise.all([
    refetchProjectApps(reactQueryClient, projectId),
    refetchScopeDomains(projectId),
    refetchScopeCustomDomains(projectId),
    refetchScopeEnvironments(projectId),
  ]);
}

// Each hook call site gets the same live query for the same arguments, so
// components reading one thing share a single subscription instead of building
// one each. An unused live query cleans itself up and restarts on next use.
export function memoByArgs<A extends Array<string | number | null>, T>(
  create: (...args: A) => T,
): (...args: A) => T {
  const cache = new Map<string, T>();
  return (...args) => {
    const key = JSON.stringify(args);
    const cached = cache.get(key);
    if (cached !== undefined) {
      return cached;
    }
    const created = create(...args);
    cache.set(key, created);
    return created;
  };
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

export function policiesQueryFor(projectId: string, appId: string, environmentId: string) {
  return (q: InitialQueryBuilder) =>
    q
      .from({ p: collection.policies })
      .where(({ p }) =>
        and(eq(p.projectId, projectId), eq(p.appId, appId), eq(p.environmentId, environmentId)),
      )
      .orderBy(({ p }) => p._order);
}

const environmentsQuery = memoByArgs((projectId: string, appIds: string) =>
  createLiveQueryCollection(environmentsQueryFor(projectId, appIds)),
);

const domainsQuery = memoByArgs((projectId: string, appId: string | null) =>
  createLiveQueryCollection(domainsQueryFor(projectId, appId)),
);

const customDomainsQuery = memoByArgs((projectId: string, appId: string | null) =>
  createLiveQueryCollection(customDomainsQueryFor(projectId, appId)),
);

function subscribeQueryCache(onChange: () => void): () => void {
  return queryClient.getQueryCache().subscribe(onChange);
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

/**
 * A live query reports only errors inside sync, so load failures are read from
 * the source collection. A failed load records its error there and emits
 * nothing, so the flag is reread on every query cache event.
 */
function useLoadState(
  source: LoadSource,
  refetchScope: (projectId: string) => Promise<unknown>,
  projectId: string,
): Pick<DeployQuery<unknown>, "isError" | "error" | "refetch"> {
  const isError = useSyncExternalStore(
    subscribeQueryCache,
    () => source.isError,
    () => false,
  );
  const refetch = useCallback(
    () => (source.isError ? source.clearError() : refetchScope(projectId)),
    [source, refetchScope, projectId],
  );
  return { isError, error: isError ? source.lastError : null, refetch };
}

const CUSTOM_DOMAIN_MUTATIONS = {
  addCustomDomain: (
    domain: { domain: string; projectId: string; appId: string; environmentId: string },
    workspaceSlug: string,
  ): Promise<unknown> =>
    collection.customDomains.insert(
      {
        ...domain,
        id: crypto.randomUUID(),
        verificationStatus: "pending",
        dnsRecords: [],
        verificationError: null,
        domainConnectProvider: null,
        domainConnectUrl: null,
        createdAt: Date.now(),
        updatedAt: null,
      },
      { metadata: { workspaceSlug } },
    ).isPersisted.promise,
  deleteCustomDomain: (customDomainId: string): void => {
    collection.customDomains.delete(customDomainId);
  },
  retryDomainVerification,
};

function liveResult<T>(
  query: { data: T[] | undefined; isLoading: boolean },
  load: Pick<DeployQuery<unknown>, "isError" | "error" | "refetch">,
): DeployQuery<T[]> {
  // A collection whose load fails still marks itself ready with no rows
  const rows = query.data ?? [];
  return {
    ...load,
    data: query.isLoading || (load.isError && rows.length === 0) ? undefined : rows,
    isLoading: query.isLoading,
    isFetching: query.isLoading,
  };
}
