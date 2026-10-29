"use client";

import type { ActiveMarker } from "@/components/charts/use-hover-owner";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  type InstanceRow,
  METRIC_IDS,
  type MetricId,
  RANGE_PRESETS,
  type RangePreset,
  environmentRefOf,
} from "@/lib/metrics/app-metrics";
import { routes } from "@/lib/navigation/routes";
import { parseAsArrayOf, parseAsString, parseAsStringLiteral, useQueryStates } from "nuqs";
import { useState } from "react";
import { useAppId, useProjectData } from "../data-provider";
import {
  type AppMetricsScope,
  useDeployMarkers,
  useInstanceEvents,
  useMetricInstances,
  useMetricSourceByInstance,
  useMetricSources,
} from "../hooks/use-app-metrics";
import { useBucketClock } from "../hooks/use-bucket-clock";
import type { FilterKey, LogsTarget } from "./instances-model";
import {
  METRICS_TABS,
  type MetricsTab,
  metricsPageModel,
  needsInstanceSeries,
  resolveEnvironment,
} from "./model";
import { type OpenReplica, openInstanceOf, podSetOf, replicaSheetModel } from "./replica-model";

const DEFAULT_RANGE: RangePreset = "24h";

const FILTER_PARAM = {
  regions: "region",
  statuses: "status",
  instances: "instance",
} as const satisfies Record<FilterKey, string>;

export function useMetricsPage() {
  const { environments, isEnvironmentsLoading, projectId } = useProjectData();
  const workspace = useWorkspaceNavigation();
  const appId = useAppId();
  const [query, setQuery] = useQueryStates(
    {
      env: parseAsString,
      range: parseAsStringLiteral(RANGE_PRESETS).withDefault(DEFAULT_RANGE),
      tab: parseAsStringLiteral(METRICS_TABS).withDefault("instances"),
      expand: parseAsStringLiteral(METRIC_IDS),
      region: parseAsArrayOf(parseAsString).withDefault([]),
      status: parseAsArrayOf(parseAsString).withDefault([]),
      instance: parseAsArrayOf(parseAsString).withDefault([]),
    },
    { history: "replace" },
  );
  const [activeMarker, setActiveMarker] = useState<ActiveMarker>(null);
  const [openReplica, setOpenReplica] = useState<OpenReplica | null>(null);

  const environment = resolveEnvironment(environments, query.env, isEnvironmentsLoading);
  const grid = useBucketClock(query.range);
  const found = environment.kind === "found" ? environment.environment : null;
  const environmentId = found?.id ?? null;
  const scope: AppMetricsScope | null = found
    ? { appId, environment: environmentRefOf(found), grid }
    : null;

  const sources = useMetricSources(scope);
  const perInstance = needsInstanceSeries(query.tab, query.expand);
  const instances = useMetricInstances(scope, perInstance);
  const byInstance = useMetricSourceByInstance(scope, {
    enabled: perInstance,
    openInstance: openInstanceOf(openReplica, environmentId, instances),
    podSet: podSetOf(instances),
  });
  const deploys = useDeployMarkers(scope);
  const events = useInstanceEvents(scope, query.tab === "instances");

  const model = metricsPageModel({
    environment,
    grid,
    tab: query.tab,
    expanded: query.expand,
    sources,
    byInstance,
    deploys,
    instances,
    events,
    filter: { regions: query.region, statuses: query.status, instances: query.instance },
  });

  const logsHref = (target: LogsTarget) =>
    routes.projects.logs({
      workspaceSlug: workspace.slug,
      projectId,
      appId,
      deploymentId: target.deploymentIds,
      instanceId: target.instanceIds,
      startTime: target.window?.startMs,
      endTime: target.window?.endMs,
    });

  return {
    model,
    environmentId,
    range: query.range,
    tab: query.tab,
    setEnvironment: (id: string) => {
      const next = environments.find((e) => e.id === id);
      if (next) {
        setQuery({ env: next.slug });
      }
    },
    setRange: (range: RangePreset) => setQuery({ range }),
    setTab: (tab: MetricsTab) => {
      setOpenReplica(null);
      setQuery({ tab });
    },
    expand: (metric: MetricId) => setQuery({ expand: metric }),
    collapse: () => setQuery({ expand: null }),
    activeMarker,
    setActiveMarker,
    setFilter: (key: FilterKey, values: string[]) =>
      setQuery({ [FILTER_PARAM[key]]: values.length > 0 ? values : null }),
    sheet: replicaSheetModel({
      open: openReplica,
      environmentId,
      instances,
      usage: byInstance,
    }),
    logsHref,
    openReplica: (row: InstanceRow) => setOpenReplica({ environmentId, row }),
    closeReplica: () => setOpenReplica(null),
  };
}

export type MetricsPageController = ReturnType<typeof useMetricsPage>;
