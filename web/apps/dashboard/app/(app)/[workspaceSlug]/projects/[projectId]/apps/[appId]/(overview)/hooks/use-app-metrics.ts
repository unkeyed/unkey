"use client";

import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { type Loadable, type QueryResult, fromQuery } from "@/lib/loadable/loadable";
import type {
  Grid,
  InstanceRow,
  InstanceSeries,
  SourceResult,
  Sources,
} from "@/lib/metrics/app-metrics";
import { trpc } from "@/lib/trpc/client";
import { useEffect, useRef, useState } from "react";

export type AppMetricsScope = { appId: string; environmentId: string; grid: Grid };

function inputOf(scope: AppMetricsScope | null) {
  return {
    appId: scope?.appId ?? "",
    environmentId: scope?.environmentId ?? "",
    range: scope?.grid.range ?? "24h",
    endMs: scope?.grid.endMs ?? 0,
  };
}

function viewKeyOf(scope: AppMetricsScope | null): string | null {
  return scope && `${scope.appId}:${scope.environmentId}:${scope.grid.range}`;
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

const CLOCK_DRIVEN = { staleTime: Number.POSITIVE_INFINITY, refetchOnWindowFocus: false };

const CLAMPED_RETRY_MS = 2_000;

// The server clamps endMs to its own clock, so a tick that lands just before the
// server reaches the boundary gets the previous bucket back; ask again shortly.
// A gap wider than one bucket is clock skew, which retrying cannot close.
function gridDriven<T extends { grid: Grid }>(scope: AppMetricsScope | null, enabled = true) {
  return {
    ...CLOCK_DRIVEN,
    enabled: enabled && scope !== null,
    refetchInterval: (data: T | undefined) => {
      if (!data || !scope) {
        return false;
      }
      const behindMs = scope.grid.endMs - data.grid.endMs;
      return behindMs > 0 && behindMs <= scope.grid.bucketMs ? CLAMPED_RETRY_MS : false;
    },
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
  const environmentId = scope?.environmentId;
  const seen = useRef<{ environmentId: string; podSet: string } | null>(null);
  useEffect(() => {
    if (podSet === null || !appId || !environmentId) {
      return;
    }
    const previous = seen.current;
    seen.current = { environmentId, podSet };
    if (previous?.environmentId === environmentId && previous.podSet !== podSet) {
      utils.deploy.appMetrics.sourceByInstance.invalidate({ appId, environmentId });
    }
  }, [podSet, appId, environmentId, utils]);
  return useHeldAcrossTicks(scope, query);
}

export function useMetricInstances(
  environment: { appId: string; environmentId: string } | null,
  enabled: boolean,
): Loadable<InstanceRow[]> {
  const active = enabled && environment !== null;
  const query = trpc.deploy.appMetrics.instances.useQuery(
    { appId: environment?.appId ?? "", environmentId: environment?.environmentId ?? "" },
    { enabled: active, refetchOnWindowFocus: false },
  );
  useCollectionPolling(query.refetch, { intervalMs: INSTANCES_POLL_MS, enabled: active });
  return fromQuery(query);
}
