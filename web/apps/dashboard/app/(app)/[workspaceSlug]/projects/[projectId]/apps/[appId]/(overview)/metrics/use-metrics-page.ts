"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  type InstanceRow,
  METRIC_IDS,
  type MetricId,
  RANGE_PRESETS,
  type RangePreset,
} from "@/lib/metrics/app-metrics";
import { routes } from "@/lib/navigation/routes";
import { parseAsString, parseAsStringLiteral, useQueryStates } from "nuqs";
import { useState } from "react";
import { useAppId, useProjectData } from "../data-provider";
import {
  type AppMetricsScope,
  useDeployMarkers,
  useMetricInstances,
  useMetricSourceByInstance,
  useMetricSources,
} from "../hooks/use-app-metrics";
import { useBucketClock } from "../hooks/use-bucket-clock";
import {
  METRICS_TABS,
  type MetricsTab,
  metricsPageModel,
  needsInstanceSeries,
  resolveEnvironment,
} from "./model";
import { type OpenReplica, openInstanceOf, podSetOf, replicaSheetModel } from "./replica-model";

const DEFAULT_RANGE: RangePreset = "24h";

export function useMetricsPage() {
  const { environments, isEnvironmentsLoading, projectId } = useProjectData();
  const workspace = useWorkspaceNavigation();
  const appId = useAppId();
  const [query, setQuery] = useQueryStates(
    {
      env: parseAsString,
      range: parseAsStringLiteral(RANGE_PRESETS).withDefault(DEFAULT_RANGE),
      tab: parseAsStringLiteral(METRICS_TABS).withDefault("traffic"),
      expand: parseAsStringLiteral(METRIC_IDS),
      region: parseAsString,
      instance: parseAsString,
    },
    { history: "replace" },
  );
  const [activeDeployId, setActiveDeployId] = useState<string | null>(null);
  const [openReplica, setOpenReplica] = useState<OpenReplica | null>(null);

  const environment = resolveEnvironment(environments, query.env, isEnvironmentsLoading);
  const grid = useBucketClock(query.range);
  const scope: AppMetricsScope | null =
    environment.kind === "found"
      ? { appId, environmentId: environment.environment.id, grid }
      : null;

  const environmentId = scope?.environmentId ?? null;
  const sources = useMetricSources(scope);
  const perInstance = needsInstanceSeries(query.tab, query.expand);
  const instances = useMetricInstances(scope, perInstance);
  const byInstance = useMetricSourceByInstance(scope, {
    enabled: perInstance,
    openInstance: openInstanceOf(openReplica, environmentId, instances),
    podSet: podSetOf(instances),
  });
  const deploys = useDeployMarkers(scope);

  const model = metricsPageModel({
    environment,
    grid,
    tab: query.tab,
    expanded: query.expand,
    sources,
    byInstance,
    deploys,
    instances,
    region: query.region,
    selected: query.instance,
  });

  const logsHref = (row: InstanceRow) =>
    routes.projects.logs({
      workspaceSlug: workspace.slug,
      projectId,
      deploymentId: row.deployment.id,
      instanceId: row.instanceId,
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
    activeDeployId,
    setActiveDeployId,
    setRegion: (region: string | null) => setQuery({ region }),
    clearInstance: () => setQuery({ instance: null }),
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
