"use client";

import { type AppMetricsRange, resolveAppMetricsRange } from "@unkey/clickhouse";
import type { DeployMarker } from "../components/metrics-chart";
import { shortSha } from "../lib/series";
import {
  type MetricsScope,
  useDeploymentMarkers,
  useLatencySeries,
  useRequestSeries,
  useResourceSeries,
} from "./use-app-metrics";

// Every series the page can show, fetched once per scope. Cards read the slice
// they need; the shared range keeps all x-axes aligned even while some queries
// are still loading.
export function useMetricsBundle(scope: MetricsScope) {
  const cpu = useResourceSeries(scope, "cpu");
  const memory = useResourceSeries(scope, "memory");
  const disk = useResourceSeries(scope, "disk");
  const egress = useResourceSeries(scope, "egress");
  const ingress = useResourceSeries(scope, "ingress");
  const requests = useRequestSeries(scope);
  const latency = useLatencySeries(scope);
  const markersQuery = useDeploymentMarkers(scope);

  const range: AppMetricsRange =
    requests.data?.range ??
    cpu.data?.range ??
    markersQuery.data?.range ??
    resolveAppMetricsRange(scope.window, Date.now());

  const markers: DeployMarker[] = (markersQuery.data?.markers ?? []).map((m) => ({
    id: m.id,
    x: m.createdAt,
    label: shortSha(m.gitCommitSha) || m.id.slice(-7),
    title: m.gitCommitMessage ?? undefined,
  }));

  return { cpu, memory, disk, egress, ingress, requests, latency, markers, range };
}

export type MetricsBundle = ReturnType<typeof useMetricsBundle>;
