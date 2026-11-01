"use client";

import { collection } from "@/lib/collections";
import { queryClient } from "@/lib/collections/client";
import {
  DEFAULT_DEPLOYMENT_STATUS_GROUPS,
  DEPLOYMENT_STATUSES,
  type DeploymentStatus,
  expandDeploymentStatusGroups,
  isDeploymentSettling,
} from "@/lib/collections/deploy/deployment-status";
import { type Deployment, withDeploymentDetails } from "@/lib/collections/deploy/deployments";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import {
  type InfiniteData,
  InfiniteQueryObserver,
  type InfiniteQueryObserverOptions,
  type QueryKey,
  hashKey,
  notifyManager,
} from "@tanstack/query-core";
import {
  type InitialQueryBuilder,
  and,
  createLiveQueryCollection,
  eq,
  inArray,
  useLiveQuery,
} from "@tanstack/react-db";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { useCallback, useEffect, useMemo, useState, useSyncExternalStore } from "react";
import { useAppScope, useDeployScope } from "./context";
import {
  type QueryResult,
  memoByArgs,
  refetchScopeDeployments,
  useApp,
  useEnvironments,
} from "./queries";

// Both time bounds are inclusive, as the time filter writes them
export type DeploymentListFilter = {
  statuses: DeploymentStatus[];
  environmentId?: string;
  branches: string[];
  startTime?: number;
  endTime?: number;
};

export type DeploymentsPageResult = {
  rows: Deployment[];
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: () => Promise<unknown>;
  refetch: () => Promise<unknown>;
};

// Pickers that offer past deployments by id or image look this far back
export const DEPLOYMENT_PICKER_LIMIT = 100;

// The app overview lists this many and treats none as an app never deployed
export const RECENT_DEPLOYMENTS_LIMIT = 5;

// The deployments list, newest first, one API cursor page at a time
export function useDeploymentsPage(filter: DeploymentListFilter): DeploymentsPageResult {
  const { projectId, appId } = useAppScope();
  const options = useMemo(() => {
    const page = deploymentsPageOptions(projectId, appId, filter);
    // The observer applies the client defaults itself; these are the fields
    // an optimistic read needs before it has
    return {
      ...page,
      queryHash: hashKey(page.queryKey),
      throwOnError: false,
      refetchOnReconnect: true,
      _optimisticResults: "optimistic" as const,
    };
  }, [projectId, appId, filter]);
  const [observer] = useState(() => new InfiniteQueryObserver(queryClient, options));
  const result = observer.getOptimisticResult(options);
  useSyncExternalStore(
    useCallback(
      (onChange: () => void) => observer.subscribe(notifyManager.batchCalls(onChange)),
      [observer],
    ),
    () => observer.getCurrentResult(),
    () => observer.getCurrentResult(),
  );
  useEffect(() => observer.setOptions(options), [observer, options]);

  const rows = useMemo(() => result.data?.pages.flatMap((page) => page.rows) ?? [], [result.data]);
  return {
    rows,
    isLoading: result.isLoading,
    isError: result.isError,
    error: result.error,
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    fetchNextPage: result.fetchNextPage,
    refetch: result.refetch,
  };
}

// A deployment of the scope by id, however old it is. Null when it does not exist.
export function useDeployment(
  deploymentId: string | null | undefined,
): QueryResult<Deployment | null | undefined> {
  const { projectId, appId } = useDeployScope();
  const refetch = useCallback(() => refetchScopeDeployments(projectId, appId), [projectId, appId]);
  const query = useLiveQuery(
    () => (deploymentId ? deploymentByIdQuery(projectId, appId ?? null, deploymentId) : null),
    [projectId, appId, deploymentId],
  );
  // A row another list of the scope already holds shows while its own load is in flight
  const held =
    query.isLoading && deploymentId ? collection.deployments.get(deploymentId) : undefined;
  const listed =
    held && held.projectId === projectId && (appId === undefined || held.appId === appId)
      ? held
      : query.isLoading && deploymentId && appId !== undefined
        ? queryClient
            .getQueriesData<InfiniteData<DeploymentsPage>>({
              queryKey: queryKeys.deployments.pages(projectId, appId),
            })
            .flatMap(([, data]) => data?.pages ?? [])
            .flatMap((page) => page.rows)
            .find((row) => row.id === deploymentId)
        : undefined;
  return {
    data: deploymentId ? (query.isLoading ? listed : (query.data?.[0] ?? null)) : null,
    isLoading: query.isLoading && listed === undefined,
    isError: query.isError,
    error: null,
    refetch,
  };
}

// The deployment serving the app, or the project's primary app outside an app
export function useCurrentDeployment({
  enabled = true,
}: { enabled?: boolean } = {}): QueryResult<Deployment | null | undefined> {
  const app = useApp();
  const currentDeploymentId = app.data?.currentDeploymentId;
  const deployment = useDeployment(enabled ? currentDeploymentId : null);
  return {
    ...deployment,
    data: app.isLoading && currentDeploymentId === undefined ? undefined : deployment.data,
    isLoading: app.isLoading || deployment.isLoading,
  };
}

// The newest deployments of the scope, skipped pushes left out
export function useRecentDeployments({
  limit,
  enabled = true,
}: {
  limit: number;
  enabled?: boolean;
}): QueryResult<Deployment[]> {
  const { projectId, appId } = useDeployScope();
  const refetch = useCallback(() => refetchScopeDeployments(projectId, appId), [projectId, appId]);
  const query = useLiveQuery(
    () => (enabled ? recentDeploymentsQuery(projectId, appId ?? null, null, limit) : null),
    [projectId, appId, limit, enabled],
  );
  return {
    data: query.data ?? [],
    isLoading: enabled && query.isLoading,
    isError: query.isError,
    error: null,
    refetch,
  };
}

/**
 * The app's newest production deployments, skipped pushes left out. Enough to
 * find the latest production deployment, rollback targets and the deployment a
 * rollback replaced.
 */
export function useProductionDeployments({
  enabled = true,
}: { enabled?: boolean } = {}): QueryResult<Deployment[]> {
  const { projectId, appId } = useAppScope();
  const environments = useEnvironments();
  const productionId = environments.data.find((e) => e.kind === ENVIRONMENT_KIND.production)?.id;
  const refetch = useCallback(() => refetchScopeDeployments(projectId, appId), [projectId, appId]);
  const query = useLiveQuery(
    () =>
      enabled && productionId
        ? recentDeploymentsQuery(projectId, appId, productionId, PRODUCTION_LIMIT)
        : null,
    [projectId, appId, productionId, enabled],
  );
  return {
    data: query.data ?? [],
    isLoading: enabled && (environments.isLoading || query.isLoading),
    isError: query.isError,
    error: null,
    refetch,
  };
}

// What a visit to the deployments list without filters in the url loads first
export function warmDeploymentsPage(projectId: string, appId: string): void {
  queryClient
    .prefetchInfiniteQuery(deploymentsPageOptions(projectId, appId, DEFAULT_FILTER))
    .catch(() => {});
}

export function deploymentByIdQueryFor(
  projectId: string,
  appId: string | null,
  deploymentId: string,
) {
  return (q: InitialQueryBuilder) =>
    q
      .from({ deployment: collection.deployments })
      .where(({ deployment }) =>
        appId
          ? and(
              eq(deployment.projectId, projectId),
              eq(deployment.appId, appId),
              eq(deployment.id, deploymentId),
            )
          : and(eq(deployment.projectId, projectId), eq(deployment.id, deploymentId)),
      );
}

export function recentDeploymentsQueryFor(
  projectId: string,
  appId: string | null,
  environmentId: string | null,
  limit: number,
) {
  return (q: InitialQueryBuilder) => {
    let query = q
      .from({ deployment: collection.deployments })
      .where(({ deployment }) =>
        and(eq(deployment.projectId, projectId), inArray(deployment.status, RECENT_STATUSES)),
      );
    if (appId) {
      query = query.where(({ deployment }) => eq(deployment.appId, appId));
    }
    if (environmentId) {
      query = query.where(({ deployment }) => eq(deployment.environmentId, environmentId));
    }
    return query.orderBy(({ deployment }) => deployment.createdAt, "desc").limit(limit);
  };
}

export const PRODUCTION_LIMIT = 25;

/** Every deployment this tab holds for the scope, in live queries or list pages. */
export function scopeDeployments(projectId: string, appId: string | undefined): Deployment[] {
  const rows = new Map<string, Deployment>();
  for (const deployment of collection.deployments.values()) {
    if (deployment.projectId === projectId && (appId === undefined || deployment.appId === appId)) {
      rows.set(deployment.id, deployment);
    }
  }
  if (appId !== undefined) {
    const lists = queryClient.getQueriesData<InfiniteData<DeploymentsPage>>({
      queryKey: queryKeys.deployments.pages(projectId, appId),
      type: "active",
    });
    for (const [, data] of lists) {
      for (const deployment of data?.pages.flatMap((page) => page.rows) ?? []) {
        rows.set(deployment.id, deployment);
      }
    }
  }
  return [...rows.values()];
}

export function subscribeDeployments(onChange: () => void): () => void {
  const subscription = collection.deployments.subscribeChanges(onChange, {
    includeInitialState: false,
  });
  const unsubscribeCache = queryClient.getQueryCache().subscribe((event) => {
    if (event.query.queryKey[0] === "deployments") {
      onChange();
    }
  });
  return () => {
    subscription.unsubscribe();
    unsubscribeCache();
  };
}

// Compared before and after a poll to tell whether any row appeared or moved
export function deploymentStates(rows: readonly Deployment[]): string {
  return rows
    .map((d) => `${d.id}:${d.status}:${isDeploymentSettling(d) ? 1 : 0}`)
    .sort()
    .join(",");
}

const PAGE_SIZE = 25;

// Skipped pushes never deployed, so the shared views leave them out
const RECENT_STATUSES = DEPLOYMENT_STATUSES.filter((status) => status !== "skipped");

// The list's first visit shows the status groups that are not hidden by default
const DEFAULT_FILTER: DeploymentListFilter = {
  statuses: expandDeploymentStatusGroups(DEFAULT_DEPLOYMENT_STATUS_GROUPS),
  branches: [],
};

type DeploymentsPage = { rows: Deployment[]; nextCursor: string | undefined };

const deploymentByIdQuery = memoByArgs(
  (projectId: string, appId: string | null, deploymentId: string) =>
    createLiveQueryCollection(deploymentByIdQueryFor(projectId, appId, deploymentId)),
);

const recentDeploymentsQuery = memoByArgs(
  (projectId: string, appId: string | null, environmentId: string | null, limit: number) =>
    createLiveQueryCollection(recentDeploymentsQueryFor(projectId, appId, environmentId, limit)),
);

function deploymentsPageOptions(
  projectId: string,
  appId: string,
  filter: DeploymentListFilter,
): InfiniteQueryObserverOptions<
  DeploymentsPage,
  Error,
  InfiniteData<DeploymentsPage>,
  QueryKey,
  string | undefined
> {
  const queryKey = queryKeys.deployments.page(projectId, appId, filter);
  return {
    queryKey,
    retry: 3,
    // A poll that changes nothing on screen does not re-render the list
    notifyOnChangeProps: [
      "data",
      "isLoading",
      "isError",
      "error",
      "hasNextPage",
      "isFetchingNextPage",
    ],
    initialPageParam: undefined,
    getNextPageParam: (page) => page.nextCursor,
    queryFn: async ({ pageParam }) => {
      let response: Awaited<
        ReturnType<ReturnType<typeof getUnkeyClient>["deployments"]["listDeployments"]>
      >;
      try {
        response = await getUnkeyClient().deployments.listDeployments({
          project: projectId,
          app: appId,
          environment: filter.environmentId,
          status: filter.statuses,
          branch: filter.branches.length > 0 ? filter.branches : undefined,
          startTime: filter.startTime,
          // The API's end bound is exclusive
          endTime: filter.endTime === undefined ? undefined : filter.endTime + 1,
          limit: PAGE_SIZE,
          cursor: pageParam,
        });
      } catch (error) {
        if (error instanceof NotFoundErrorResponse) {
          return { rows: [], nextCursor: undefined };
        }
        throw error;
      }
      const { data, pagination } = response.result;
      const held = new Map(
        queryClient
          .getQueryData<InfiniteData<DeploymentsPage>>(queryKey)
          ?.pages.flatMap((page) => page.rows.map((row) => [row.id, row] as const)),
      );
      const rows = await withDeploymentDetails(
        data,
        projectId,
        (id) => held.get(id) ?? collection.deployments.get(id),
      );
      return { rows, nextCursor: pagination.hasMore ? pagination.cursor : undefined };
    },
  };
}
