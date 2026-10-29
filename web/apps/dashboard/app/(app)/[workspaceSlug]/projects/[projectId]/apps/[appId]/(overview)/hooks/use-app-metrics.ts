"use client";

import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { type Loadable, type QueryResult, fromQuery } from "@/lib/loadable/loadable";
import {
  type EnvironmentRef,
  type Grid,
  type InstanceEvent,
  type InstanceRow,
  type InstanceSeries,
  type SourceResult,
  type Sources,
  clampedRetryMs,
  environmentKeyOf,
} from "@/lib/metrics/app-metrics";
import { trpc } from "@/lib/trpc/client";
import { useEffect, useRef, useState } from "react";

export type AppMetricsScope = { appId: string; environment: EnvironmentRef; grid: Grid };

function inputOf(scope: AppMetricsScope | null) {
  return {
    appId: scope?.appId ?? "",
    environment: scope?.environment ?? { id: "" },
    range: scope?.grid.range ?? "24h",
    endMs: scope?.grid.endMs ?? 0,
  };
}

function viewKeyOf(scope: AppMetricsScope | null): string | null {
  return scope && `${scope.appId}:${environmentKeyOf(scope.environment)}:${scope.grid.range}`;
}

// Holds the last result while only endMs or the open pod moved, so charts do not
// flash to loading every bucket or rollout. A new app, environment or range
// shows loading instead of the previous view's data.
function useHeldAcrossTicks<T>(scope: AppMetricsScope | null, query: QueryResult<T>): Loadable<T> {
  const viewKey = viewKeyOf(scope);
  const [held, setHeld] = useState<{ viewKey: string | null; data: T } | null>(null);
  if (query.data !== undefined && (held?.data !== query.data || held.viewKey !== viewKey)) {
    setHeld({ viewKey, data: query.data });
  }
  const data = query.data ?? (held && held.viewKey === viewKey ? held.data : undefined);
  return fromQuery({ ...query, data });
}

const CLOCK_DRIVEN = {
  staleTime: Number.POSITIVE_INFINITY,
  refetchOnWindowFocus: false,
  // Unbatched so a fast source never waits on a slow one
  trpc: { context: { skipBatch: true } },
};

function gridDriven<T extends { grid: Grid }>(scope: AppMetricsScope | null, enabled = true) {
  return {
    ...CLOCK_DRIVEN,
    enabled: enabled && scope !== null,
    refetchInterval: (
      data: T | undefined,
      { state }: { state: { dataUpdateCount: number; errorUpdateCount: number } },
    ) =>
      data && scope
        ? clampedRetryMs(scope.grid, data.grid, state.dataUpdateCount + state.errorUpdateCount)
        : false,
  };
}

const INSTANCES_POLL_MS = 10_000;

export function useMetricSources(scope: AppMetricsScope | null): Sources {
  const traffic = trpc.deploy.appMetrics.traffic.useQuery(
    inputOf(scope),
    gridDriven<SourceResult<"traffic">>(scope),
  );
  const resources = trpc.deploy.appMetrics.resources.useQuery(
    inputOf(scope),
    gridDriven<SourceResult<"resources">>(scope),
  );
  return {
    traffic: useHeldAcrossTicks(scope, traffic),
    resources: useHeldAcrossTicks(scope, resources),
  };
}

export function useDeployMarkers(scope: AppMetricsScope | null) {
  const query = trpc.deploy.appMetrics.deploys.useQuery(inputOf(scope), {
    ...CLOCK_DRIVEN,
    enabled: scope !== null,
  });
  return useHeldAcrossTicks(scope, query);
}

export function useInstanceEvents(
  scope: AppMetricsScope | null,
  enabled: boolean,
): Loadable<InstanceEvent[]> {
  const query = trpc.deploy.appMetrics.instanceEvents.useQuery(inputOf(scope), {
    ...CLOCK_DRIVEN,
    enabled: enabled && scope !== null,
  });
  return useHeldAcrossTicks(scope, query);
}

export function useMetricSourceByInstance(
  scope: AppMetricsScope | null,
  {
    enabled,
    openInstance,
    podSet,
  }: { enabled: boolean; openInstance: string | null; podSet: string | null },
) {
  const query = trpc.deploy.appMetrics.sourceByInstance.useQuery(
    { ...inputOf(scope), openInstance },
    gridDriven<InstanceSeries>(scope, enabled),
  );
  const utils = trpc.useUtils();
  const appId = scope?.appId;
  const environment = scope?.environment;
  const seen = useRef<{ environmentKey: string; podSet: string } | null>(null);
  useEffect(() => {
    if (podSet === null || !appId || !environment) {
      return;
    }
    const environmentKey = environmentKeyOf(environment);
    const previous = seen.current;
    seen.current = { environmentKey, podSet };
    if (previous?.environmentKey === environmentKey && previous.podSet !== podSet) {
      utils.deploy.appMetrics.sourceByInstance.invalidate({ appId, environment });
    }
  }, [podSet, appId, environment, utils]);
  return useHeldAcrossTicks(scope, query);
}

export function useMetricInstances(
  scope: AppMetricsScope | null,
  enabled: boolean,
): Loadable<InstanceRow[]> {
  const active = enabled && scope !== null;
  const query = trpc.deploy.appMetrics.instances.useQuery(
    { appId: scope?.appId ?? "", environment: scope?.environment ?? { id: "" } },
    { enabled: active, refetchOnWindowFocus: false },
  );
  useCollectionPolling(query.refetch, { intervalMs: INSTANCES_POLL_MS, enabled: active });
  return fromQuery(query);
}
