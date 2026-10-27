import type { AreaChartPoint } from "@/components/charts/area-timeseries";
import type { ChartMarker } from "@/components/charts/chart-markers";
import type { Environment } from "@/lib/collections/deploy/environments";
import type { Loadable } from "@/lib/loadable/loadable";
import type {
  DeployMarker,
  Grid,
  InstanceRow,
  InstanceSeries,
  MetricId,
  Sources,
} from "@/lib/metrics/app-metrics";
import { type ChartState, METRIC_VIEWS, cardState } from "@/lib/metrics/views";
import { match } from "@unkey/match";
import { type DeployEffect, type EffectSamples, deployEffects } from "./deploy-effects";
import {
  INSTANCE_METRICS,
  type InstanceChart,
  type InstanceMetric,
  type InstancesTabView,
  type RegionFilter,
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

export const METRICS_TABS = ["traffic", "resources", "instances"] as const;
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

export type MetricCardModel = { metric: MetricId; state: ChartState };

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
  deploys: DeployView[];
};

export type MetricsPageModel =
  | { kind: "loading" }
  | { kind: "no-environment" }
  | {
      kind: "ready";
      environment: MetricsEnvironment;
      tab: MetricsTab;
      view: TabView;
      regionFilter: RegionFilter | null;
      markers: ChartMarker[];
      deploys: ReadonlyMap<string, DeployView>;
      panel: PanelModel | null;
    };

function readyOrNull<T>(loadable: Loadable<T>): T | null {
  return loadable.status === "ready" ? loadable.data : null;
}

function effectSamples({ traffic, resources }: Sources): EffectSamples {
  return { traffic: readyOrNull(traffic), resources: readyOrNull(resources) };
}

function cardOf(sources: Sources, metric: MetricId): MetricCardModel {
  return { metric, state: cardState(sources, metric) };
}

type MetricsPageInput = {
  environment: EnvironmentResolution;
  grid: Grid;
  tab: MetricsTab;
  expanded: MetricId | null;
  sources: Sources;
  byInstance: Loadable<InstanceSeries>;
  deploys: Loadable<DeployMarker[]>;
  instances: Loadable<InstanceRow[]>;
  region: string | null;
  selected: string | null;
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
          hero: cardOf(sources, layout.hero),
          grid: layout.grid.map((metric) => cardOf(sources, metric)),
        }
      : {
          kind: "instances",
          instances: instancesTabView({
            instances: input.instances,
            usage: input.byInstance,
            region: input.region,
            selected: input.selected,
          }),
        };

  return {
    kind: "ready",
    environment,
    tab: input.tab,
    view,
    regionFilter: regionFilterOf(view),
    markers: markers.map((d) => ({
      id: d.deploymentId,
      atMs: d.readyAtMs,
      span: d.serving && { endMs: d.serving.untilMs },
    })),
    deploys: new Map(deploys.map((d) => [d.marker.deploymentId, d])),
    panel: input.expanded ? panelOf(input.expanded, input, deploys) : null,
  };
}

function regionFilterOf(view: TabView): RegionFilter | null {
  if (view.kind !== "instances" || view.instances.status !== "ready") {
    return null;
  }
  const { regions, region } = view.instances;
  return { regions, region };
}

function panelOf(
  metric: MetricId,
  input: {
    grid: Grid;
    sources: Sources;
    byInstance: Loadable<InstanceSeries>;
    instances: Loadable<InstanceRow[]>;
  },
  deploys: DeployView[],
): PanelModel {
  const card = cardOf(input.sources, metric);
  return {
    card,
    stats:
      card.state.status === "ready"
        ? panelStats(card.state.chart, METRIC_VIEWS[metric].primary)
        : null,
    byInstance: isInstanceMetric(metric)
      ? instanceChart(input.byInstance, input.instances, metric)
      : null,
    deploys: deploys.filter((d) => d.marker.readyAtMs >= input.grid.startMs).reverse(),
  };
}
