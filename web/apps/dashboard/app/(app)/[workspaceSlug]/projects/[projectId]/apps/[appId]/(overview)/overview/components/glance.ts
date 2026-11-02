import {
  METRIC_CATALOG,
  type MetricSource,
  type SourcePoint,
  type Sources,
} from "@/lib/metrics/app-metrics";
import {
  type ChartState,
  type MetricUnit,
  type SourceOf,
  cardState,
  errorRate,
  sumOf,
} from "@/lib/metrics/views";

export const GLANCE_METRICS = ["requests", "errors", "cpu"] as const;

export type GlanceMetric = (typeof GLANCE_METRICS)[number];

type Headline<S extends MetricSource> = {
  label: string;
  unit: MetricUnit;
  reduce: (points: ReadonlyArray<SourcePoint<S>>, bucketSec: number) => number | null;
};

export const GLANCE: { [M in GlanceMetric]: Headline<SourceOf<M>> } = {
  requests: {
    label: "Total",
    unit: "count",
    reduce: (points, bucketSec) => sumOf(points, (p) => p.requests * bucketSec) || null,
  },
  errors: { label: "5xx rate", unit: "ratio", reduce: errorRate },
  cpu: {
    label: "Latest",
    unit: "millicores",
    reduce: (points) => points.findLast((p) => p.cpu !== null)?.cpu ?? null,
  },
};

export function glanceValue<M extends GlanceMetric>(metric: M, sources: Sources): number | null {
  const source: SourceOf<M> = METRIC_CATALOG[metric].source;
  const headline: Headline<SourceOf<M>> = GLANCE[metric];
  const loaded = sources[source];
  return loaded.status === "ready"
    ? headline.reduce(loaded.data.points, loaded.data.grid.bucketMs / 1000)
    : null;
}

type ReadyChart = Extract<ChartState, { status: "ready" }>;

export type GlanceCellState = Exclude<ChartState, ReadyChart> | (ReadyChart & { headline: number });

function hasSignal(chart: ReadyChart["chart"]): boolean {
  return chart.some((point) =>
    Object.entries(point).some(([key, value]) => key !== "originalTimestamp" && (value ?? 0) > 0),
  );
}

export function glanceCellState(metric: GlanceMetric, sources: Sources): GlanceCellState {
  const state = cardState(sources, metric);
  if (state.status !== "ready") {
    return state;
  }
  const headline = glanceValue(metric, sources);
  return headline === null || !hasSignal(state.chart)
    ? { status: "empty" }
    : { ...state, headline };
}
