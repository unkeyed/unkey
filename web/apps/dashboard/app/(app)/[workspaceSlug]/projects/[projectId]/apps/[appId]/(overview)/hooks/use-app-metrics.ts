"use client";

import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { type Loadable, type QueryResult, fromQuery } from "@/lib/loadable/loadable";
import type { Grid, InstanceRow, Sources } from "@/lib/metrics/app-metrics";
import { trpc } from "@/lib/trpc/client";
import { useState } from "react";

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

// Holds the last result across a clock tick (only endMs moved) so charts do not
// flash to loading every bucket. Any other change shows loading instead of the
// previous environment's or range's data.
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

const INSTANCES_POLL_MS = 10_000;

export function useMetricSources(scope: AppMetricsScope | null): Sources {
  const options = { ...CLOCK_DRIVEN, enabled: scope !== null };
  const traffic = trpc.deploy.appMetrics.traffic.useQuery(inputOf(scope), options);
  const resources = trpc.deploy.appMetrics.resources.useQuery(inputOf(scope), options);
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

export function useMetricSourceByInstance(scope: AppMetricsScope | null, enabled: boolean) {
  const query = trpc.deploy.appMetrics.sourceByInstance.useQuery(inputOf(scope), {
    ...CLOCK_DRIVEN,
    enabled: enabled && scope !== null,
  });
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
