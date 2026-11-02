"use client";

import {
  DEFAULT_DEPLOYMENT_STATUS_GROUPS,
  DEPLOYMENT_STATUSES,
  type DeploymentStatus,
  expandDeploymentStatusGroups,
  isDeploymentSettling,
} from "@/lib/collections/deploy/deployment-status";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { type Deployment, withDeploymentDetails } from "@/lib/deploy-data/deployment";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import {
  type InfiniteData,
  type QueryClient,
  type UseQueryResult,
  useInfiniteQuery,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import type { Deployment as ApiDeployment } from "@unkey/api/models/components";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { useMemo, useSyncExternalStore } from "react";
import { useAppScope, useDeployScope, useScopeStore } from "./context";
import { type DeployQuery, useApp, useEnvironments } from "./queries";

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

export const PRODUCTION_LIMIT = 25;

// The deployments list, newest first, one API cursor page at a time
export function useDeploymentsPage(filter: DeploymentListFilter): DeploymentsPageResult {
  const { projectId, appId } = useAppScope();
  const queryClient = useQueryClient();
  const refetchInterval = usePollInterval((data: InfiniteData<DeploymentsPage> | undefined) =>
    (data?.pages ?? []).flatMap((page) => page.rows),
  );
  const query = useInfiniteQuery({
    ...deploymentListOptions(queryClient, projectId, appId, filter),
    refetchInterval,
  });
  const rows = useMemo(() => query.data?.pages.flatMap((page) => page.rows) ?? [], [query.data]);
  return {
    rows,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    hasNextPage: query.hasNextPage ?? false,
    isFetchingNextPage: query.isFetchingNextPage,
    fetchNextPage: query.fetchNextPage,
    refetch: query.refetch,
  };
}

// A deployment of the scope by id, however old it is. Null when it does not
// exist or belongs to another app than the scope's.
export function useDeployment(
  deploymentId: string | null | undefined,
): DeployQuery<Deployment | null> {
  const { projectId, appId } = useDeployScope();
  const queryClient = useQueryClient();
  const refetchInterval = usePollInterval((data: Deployment | null | undefined) =>
    data ? [data] : [],
  );
  const query = useQuery({
    ...deploymentDetailOptions(queryClient, projectId, deploymentId ?? ""),
    enabled: Boolean(deploymentId),
    refetchInterval,
  });
  const inScope = !query.data || appId === undefined || query.data.appId === appId;
  return {
    data: deploymentId && inScope ? query.data : null,
    isLoading: query.isInitialLoading,
    isFetching: query.isFetching,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

// The deployment serving the app, or the project's primary app outside an app
export function useCurrentDeployment({
  enabled = true,
}: { enabled?: boolean } = {}): DeployQuery<Deployment | null> {
  const app = useApp();
  const currentDeploymentId = app.data?.currentDeploymentId;
  const deployment = useDeployment(enabled ? currentDeploymentId : null);
  return {
    ...deployment,
    data: app.data === undefined ? undefined : deployment.data,
    isLoading: app.isLoading || deployment.isLoading,
    isError: app.isError || deployment.isError,
    error: app.error ?? deployment.error,
  };
}

// The newest deployments of the scope, skipped pushes left out
export function useRecentDeployments({
  limit,
  enabled = true,
}: {
  limit: number;
  enabled?: boolean;
}): DeployQuery<Deployment[]> {
  const { projectId, appId } = useDeployScope();
  const queryClient = useQueryClient();
  const refetchInterval = usePollInterval((data: Deployment[] | undefined) => data ?? []);
  const query = useQuery({
    ...recentDeploymentsOptions(queryClient, projectId, appId, null, limit),
    enabled,
    refetchInterval,
  });
  return listResult(query, query.isInitialLoading);
}

/**
 * The app's newest production deployments, skipped pushes left out. Enough to
 * find the latest production deployment, rollback targets and the deployment a
 * rollback replaced.
 */
export function useProductionDeployments({
  enabled = true,
}: { enabled?: boolean } = {}): DeployQuery<Deployment[]> {
  const { projectId, appId } = useAppScope();
  const queryClient = useQueryClient();
  const environments = useEnvironments();
  const productionId = environments.data?.find((e) => e.kind === ENVIRONMENT_KIND.production)?.id;
  const refetchInterval = usePollInterval((data: Deployment[] | undefined) => data ?? []);
  const query = useQuery({
    ...recentDeploymentsOptions(
      queryClient,
      projectId,
      appId,
      productionId ?? null,
      PRODUCTION_LIMIT,
    ),
    enabled: enabled && productionId !== undefined,
    refetchInterval,
  });
  return listResult(query, enabled && (environments.isLoading || query.isInitialLoading));
}

// What a visit to the deployments list without filters in the url loads first
export function warmDeploymentsPage(queryClient: QueryClient, projectId: string, appId: string) {
  void queryClient.prefetchInfiniteQuery(
    deploymentListOptions(queryClient, projectId, appId, DEFAULT_FILTER),
  );
}

export function deploymentDetailOptions(
  queryClient: QueryClient,
  projectId: string,
  deploymentId: string,
) {
  return {
    queryKey: queryKeys.deployments.detail(projectId, deploymentId),
    staleTime: STALE_MS,
    retry: 3,
    queryFn: async ({ signal }: { signal?: AbortSignal }): Promise<Deployment | null> => {
      let row: Awaited<
        ReturnType<ReturnType<typeof getUnkeyClient>["deployments"]["getDeployment"]>
      >;
      try {
        row = await getUnkeyClient().deployments.getDeployment({ deploymentId }, { signal });
      } catch (error) {
        if (error instanceof NotFoundErrorResponse) {
          return null;
        }
        throw error;
      }
      const [deployment] = await withDeploymentDetails([row.data], projectId, (id) =>
        cachedDeployment(queryClient, projectId, id),
      );
      return deployment ?? null;
    },
  };
}

export function recentDeploymentsOptions(
  queryClient: QueryClient,
  projectId: string,
  appId: string | undefined,
  environmentId: string | null,
  limit: number,
) {
  return {
    queryKey: queryKeys.deployments.recent(projectId, appId, environmentId, limit),
    staleTime: STALE_MS,
    retry: 3,
    queryFn: async ({ signal }: { signal?: AbortSignal }): Promise<Deployment[]> => {
      const { data } = await listDeployments(signal, {
        project: projectId,
        app: appId,
        environment: environmentId ?? undefined,
        status: RECENT_STATUSES,
        limit,
      });
      return withCachedDetails(queryClient, projectId, data);
    },
  };
}

const PAGE_SIZE = 25;

// Matches the client the collections use, so a revisit within it reads the cache
const STALE_MS = 30_000;

// Skipped pushes never deployed, so the overview and pickers leave them out
const RECENT_STATUSES = DEPLOYMENT_STATUSES.filter((status) => status !== "skipped");

// The list's first visit shows the status groups that are not hidden by default
const DEFAULT_FILTER: DeploymentListFilter = {
  statuses: expandDeploymentStatusGroups(DEFAULT_DEPLOYMENT_STATUS_GROUPS),
  branches: [],
};

type DeploymentsPage = { rows: Deployment[]; nextCursor: string | undefined };

type ListDeploymentsRequest = Parameters<
  ReturnType<typeof getUnkeyClient>["deployments"]["listDeployments"]
>[0];

// A project or app deleted under the page lists nothing rather than failing
async function listDeployments(signal: AbortSignal | undefined, request: ListDeploymentsRequest) {
  try {
    const response = await getUnkeyClient().deployments.listDeployments(request, { signal });
    return response.result;
  } catch (error) {
    if (error instanceof NotFoundErrorResponse) {
      return { data: [], pagination: { hasMore: false, cursor: undefined } };
    }
    throw error;
  }
}

function deploymentListOptions(
  queryClient: QueryClient,
  projectId: string,
  appId: string,
  filter: DeploymentListFilter,
) {
  return {
    queryKey: queryKeys.deployments.list(projectId, appId, filter),
    staleTime: STALE_MS,
    retry: 3,
    getNextPageParam: (page: DeploymentsPage) => page.nextCursor,
    queryFn: async ({
      pageParam,
      signal,
    }: { pageParam?: string; signal?: AbortSignal }): Promise<DeploymentsPage> => {
      const { data, pagination } = await listDeployments(signal, {
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
      const rows = await withCachedDetails(queryClient, projectId, data);
      return { rows, nextCursor: pagination.hasMore ? pagination.cursor : undefined };
    },
  };
}

function cachedDeployment(
  queryClient: QueryClient,
  projectId: string,
  deploymentId: string,
): Deployment | undefined {
  return (
    queryClient.getQueryData<Deployment | null>(
      queryKeys.deployments.detail(projectId, deploymentId),
    ) ?? undefined
  );
}

// Every listed row is also cached by id, so opening one reads it without a
// request, and a row already cached keeps details nothing could have changed
async function withCachedDetails(
  queryClient: QueryClient,
  projectId: string,
  rows: ApiDeployment[],
): Promise<Deployment[]> {
  const deployments = await withDeploymentDetails(rows, projectId, (id) =>
    cachedDeployment(queryClient, projectId, id),
  );
  for (const deployment of deployments) {
    queryClient.setQueryData(queryKeys.deployments.detail(projectId, deployment.id), deployment);
  }
  return deployments;
}

// When each query's data last changed, so the polls of a list that stays the same back off
const lastChange = new WeakMap<object, { data: unknown; atMs: number }>();

// Each deployments query polls on its own: faster while a row it holds is
// settling, at a page's requested cadence, and slower the longer nothing changes
function usePollInterval<T>(rowsOf: (data: T | undefined) => readonly Deployment[]) {
  const store = useScopeStore();
  const requestedMs = useSyncExternalStore(
    store.subscribe,
    store.requestedPollMs,
    store.requestedPollMs,
  );
  return (data: T | undefined, query: object) => {
    const nowMs = Date.now();
    const seen = lastChange.get(query);
    const changedAtMs = seen && seen.data === data ? seen.atMs : nowMs;
    lastChange.set(query, { data, atMs: changedAtMs });
    return deploymentsPollIntervalMs({
      settling: rowsOf(data).some(isDeploymentSettling),
      requestedMs,
      msSinceChange: nowMs - changedAtMs,
    });
  };
}

const POLL_IDLE_MS = 20_000;
const POLL_IDLE_MAX_MS = 60_000;
const POLL_SETTLING_MS = 5_000;

// Faster while a deployment of the query is settling. A page may ask for a
// faster idle cadence. Idle polls back off the longer nothing changes, since a
// page left open mostly sees no deploys: 20s, then 40s, then 60s.
export function deploymentsPollIntervalMs({
  settling,
  requestedMs,
  msSinceChange,
}: {
  settling: boolean;
  requestedMs: number | null;
  msSinceChange: number;
}): number {
  const idleMs =
    msSinceChange < POLL_IDLE_MS
      ? POLL_IDLE_MS
      : msSinceChange < POLL_IDLE_MS * 3
        ? POLL_IDLE_MS * 2
        : POLL_IDLE_MAX_MS;
  return Math.min(settling ? POLL_SETTLING_MS : idleMs, requestedMs ?? idleMs);
}

// Undefined until the first load or after it fails. A disabled query lists nothing.
function listResult(
  query: UseQueryResult<Deployment[]>,
  isLoading: boolean,
): DeployQuery<Deployment[]> {
  return {
    data: query.data ?? (isLoading || query.isError ? undefined : []),
    isLoading,
    isFetching: query.isFetching,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}
