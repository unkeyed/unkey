import type { AreaChartPoint } from "@/components/charts/area-timeseries";
import type { ChartMarker } from "@/components/charts/chart-markers";
import type { Environment } from "@/lib/collections/deploy/environments";
import type { Loadable } from "@/lib/loadable/loadable";
import { METRIC_CATALOG, isOutdated } from "@/lib/metrics/app-metrics";
import type {
  DeployMarker,
  Grid,
  HeldLoadable,
  HeldSources,
  InstanceEvent,
  InstanceRow,
  InstanceSeries,
  MetricId,
} from "@/lib/metrics/app-metrics";
import { type ChartState, METRIC_VIEWS, cardState } from "@/lib/metrics/views";
import { match } from "@unkey/match";
import { type DeployEffect, type EffectSamples, deployEffects } from "./deploy-effects";
import {
  type FilterSelection,
  INSTANCE_METRICS,
  type InstanceChart,
  type InstanceMetric,
  type InstancesTabView,
  instanceChart,
  instancesTabView,
} from "./instances-model";

export type MetricsEnvironment = Pick<Environment, "id" | "slug" | "kind">;

export type EnvironmentResolution =
  | { kind: "loading" }
  | { kind: "none" }
  | { kind: "found"; environment: MetricsEnvironment };

export function resolveEnvironment(
  environments: ReadonlyArray<MetricsEnvironment>,
  slug: string | null,
  isLoading: boolean,
): EnvironmentResolution {
  const environment =
    environments.find((e) => e.slug === slug) ??
    environments.find((e) => e.kind === "production") ??
    environments.at(0);
  if (environment) {
    return { kind: "found", environment };
  }
  return isLoading ? { kind: "loading" } : { kind: "none" };
}

export const METRICS_TABS = ["instances", "traffic", "resources"] as const;
export type MetricsTab = (typeof METRICS_TABS)[number];

type TabLayout =
  | { kind: "metrics"; label: string; hero: MetricId; grid: MetricId[] }
  | { kind: "instances"; label: string };

export const TABS: Record<MetricsTab, TabLayout> = {
  traffic: { kind: "metrics", label: "Traffic", hero: "requests", grid: ["errors", "latency"] },
  resources: {
    kind: "metrics",
    label: "Resources",
    hero: "cpu",
    grid: ["memory", "network", "disk", "instances"],
  },
  instances: { kind: "instances", label: "Instances" },
};

function isInstanceMetric(metric: MetricId): metric is InstanceMetric {
  return INSTANCE_METRICS.some((m) => m === metric);
}

export function needsInstanceSeries(tab: MetricsTab, expanded: MetricId | null): boolean {
  return tab === "instances" || (expanded !== null && isInstanceMetric(expanded));
}

export type MetricCardModel = { metric: MetricId; state: ChartState; outdated: boolean };

export function legendReadout(
  chart: ReadonlyArray<AreaChartPoint>,
  keys: ReadonlyArray<string>,
  hoveredMs: number | null,
): Record<string, number | null> | null {
  const hovered =
    hoveredMs === null ? undefined : chart.find((p) => p.originalTimestamp === hoveredMs);
  const reported = chart.some((p) => keys.some((key) => p[key] !== undefined));
  const point = hovered ?? (reported ? chart.at(-1) : undefined);
  if (!point) {
    return null;
  }
  return Object.fromEntries(keys.map((key) => [key, point[key] ?? null]));
}

export type PanelStats = { latest: number; average: number; peak: number; low: number };

export function panelStats(chart: ReadonlyArray<AreaChartPoint>, key: string): PanelStats | null {
  const values = chart.flatMap((p) => (p[key] === undefined ? [] : [p[key]]));
  const latest = values.at(-1);
  if (latest === undefined) {
    return null;
  }
  return {
    latest,
    average: values.reduce((sum, v) => sum + v, 0) / values.length,
    peak: Math.max(...values),
    low: Math.min(...values),
  };
}

export type DeployView = { marker: DeployMarker; effects: DeployEffect[] };

export type TabView =
  | { kind: "metrics"; hero: MetricCardModel; grid: MetricCardModel[] }
  | { kind: "instances"; instances: InstancesTabView };

export type PanelModel = {
  card: MetricCardModel;
  stats: PanelStats | null;
  byInstance: InstanceChart | null;
  historyTruncated: boolean;
  deploys: DeployView[];
  outdated: boolean;
};

export type MetricsPageModel =
  | { kind: "loading" }
  | { kind: "no-environment" }
  | {
      kind: "ready";
      environment: MetricsEnvironment;
      tab: MetricsTab;
      view: TabView;
      markers: ChartMarker[];
      markersOutdated: boolean;
      deploys: ReadonlyMap<string, DeployView>;
      panel: PanelModel | null;
    };

function readyOrNull<T>(loadable: Loadable<T>): T | null {
  return loadable.status === "ready" ? loadable.data : null;
}

function currentOrNull<T>(loadable: HeldLoadable<T>): T | null {
  return loadable.status === "ready" && !loadable.outdated ? loadable.data : null;
}

function effectSamples({ traffic, resources }: HeldSources): EffectSamples {
  return { traffic: currentOrNull(traffic), resources: currentOrNull(resources) };
}

function cardOf(sources: HeldSources, metric: MetricId, markersOutdated: boolean): MetricCardModel {
  const state = cardState(sources, metric);
  return {
    metric,
    state,
    outdated:
      isOutdated(sources[METRIC_CATALOG[metric].source]) ||
      (state.status === "ready" && markersOutdated),
  };
}

type MetricsPageInput = {
  environment: EnvironmentResolution;
  grid: Grid;
  tab: MetricsTab;
  expanded: MetricId | null;
  sources: HeldSources;
  byInstance: HeldLoadable<InstanceSeries>;
  deploys: HeldLoadable<DeployMarker[]>;
  instances: Loadable<InstanceRow[]>;
  events: HeldLoadable<InstanceEvent[]>;
  filter: FilterSelection;
};

export function metricsPageModel(input: MetricsPageInput): MetricsPageModel {
  return match(input.environment)
    .returnType<MetricsPageModel>()
    .with({ kind: "loading" }, () => ({ kind: "loading" }))
    .with({ kind: "none" }, () => ({ kind: "no-environment" }))
    .with({ kind: "found" }, ({ environment }) => readyModel(input, environment))
    .exhaustive();
}

function readyModel(input: MetricsPageInput, environment: MetricsEnvironment): MetricsPageModel {
  const { sources } = input;
  const markersOutdated = isOutdated(input.deploys);
  const markers = readyOrNull(input.deploys) ?? [];
  const effects = deployEffects(markers, effectSamples(sources));
  const deploys = markers.map((marker) => ({
    marker,
    effects: effects.get(marker.deploymentId) ?? [],
  }));

  const layout = TABS[input.tab];
  const view: TabView =
    layout.kind === "metrics"
      ? {
          kind: "metrics",
          hero: cardOf(sources, layout.hero, markersOutdated),
          grid: layout.grid.map((metric) => cardOf(sources, metric, markersOutdated)),
        }
      : {
          kind: "instances",
          instances: instancesTabView({
            grid: input.grid,
            instances: input.instances,
            usage: input.byInstance,
            events: input.events,
            filter: input.filter,
            markersOutdated,
          }),
        };

  return {
    kind: "ready",
    environment,
    tab: input.tab,
    view,
    markers: markers.map((d) => ({
      id: d.deploymentId,
      atMs: d.readyAtMs,
      span: d.serving && { endMs: d.serving.untilMs },
    })),
    markersOutdated,
    deploys: new Map(deploys.map((d) => [d.marker.deploymentId, d])),
    panel: input.expanded ? panelOf(input.expanded, input, deploys, markersOutdated) : null,
  };
}

function panelOf(
  metric: MetricId,
  input: {
    grid: Grid;
    sources: HeldSources;
    byInstance: HeldLoadable<InstanceSeries>;
    instances: Loadable<InstanceRow[]>;
  },
  deploys: DeployView[],
  markersOutdated: boolean,
): PanelModel {
  const card = cardOf(input.sources, metric, markersOutdated);
  const grid = card.state.status === "ready" ? card.state.grid : input.grid;
  const byInstance = isInstanceMetric(metric)
    ? instanceChart(input.byInstance, input.instances, metric)
    : null;
  return {
    card,
    historyTruncated:
      input.byInstance.status === "ready" && input.byInstance.data.history.truncated,
    stats:
      card.state.status === "ready"
        ? panelStats(card.state.chart, METRIC_VIEWS[metric].primary)
        : null,
    byInstance,
    deploys: deploys.filter((d) => d.marker.readyAtMs >= grid.startMs).reverse(),
    outdated:
      card.outdated ||
      (byInstance !== null &&
        (isOutdated(input.byInstance) || (byInstance.chart.status === "ready" && markersOutdated))),
  };
}
