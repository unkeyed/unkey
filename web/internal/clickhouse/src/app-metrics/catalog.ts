import { type Grid, findRollup } from "./grid";
import { RESOURCE_ROLLUPS, type ResourcePoint } from "./resources";
import { TRAFFIC_ROLLUPS, type TrafficPoint } from "./traffic";

type SourcePoints = { traffic: TrafficPoint; resources: ResourcePoint };
export type MetricSource = keyof SourcePoints;
export type SourcePoint<S extends MetricSource> = SourcePoints[S];
export type SourceSeriesKey<S extends MetricSource> = Exclude<keyof SourcePoints[S], "t">;

type CatalogEntry = {
  [S in MetricSource]: {
    source: S;
    series: readonly [SourceSeriesKey<S>, ...SourceSeriesKey<S>[]];
  };
}[MetricSource];

export const METRIC_IDS = [
  "requests",
  "errors",
  "latency",
  "cpu",
  "memory",
  "disk",
  "network",
  "instances",
] as const;
export type MetricId = (typeof METRIC_IDS)[number];

export const METRIC_CATALOG = {
  requests: { source: "traffic", series: ["requests"] },
  errors: { source: "traffic", series: ["errorRate"] },
  latency: { source: "traffic", series: ["p50", "p95", "p99"] },
  cpu: { source: "resources", series: ["cpu"] },
  memory: { source: "resources", series: ["memory"] },
  disk: { source: "resources", series: ["disk"] },
  network: { source: "resources", series: ["ingress", "egress"] },
  instances: { source: "resources", series: ["instances"] },
} as const satisfies Record<MetricId, CatalogEntry>;

export type SeriesKey<M extends MetricId> = (typeof METRIC_CATALOG)[M]["series"][number];

export function rollupsServe(grid: Grid, nowMs: number): boolean {
  return [TRAFFIC_ROLLUPS, RESOURCE_ROLLUPS].every(
    (rollups) => findRollup(rollups, grid, nowMs) !== null,
  );
}
