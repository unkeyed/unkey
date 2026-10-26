import type { AreaChartPoint, ValueParts } from "@/components/charts/area-timeseries";
import { formatCompactQuantity } from "@/lib/fmt";
import type { Failure, Loadable } from "@/lib/loadable/loadable";
import {
  formatBytesPerSecondParts,
  formatNetworkBytesParts,
  formatTooltipPercent,
  joinParts,
} from "@/lib/utils/deployment-formatters";
import { formatLatency } from "@/lib/utils/metric-formatters";
import {
  type Grid,
  METRIC_CATALOG,
  type MetricId,
  type RangePreset,
  type SeriesKey,
  type SourcePoint,
  type SourceResult,
  type Sources,
  type TrafficPoint,
} from "./app-metrics";

export const RANGE_LABELS: Record<RangePreset, string> = {
  "1h": "Last hour",
  "6h": "Last 6 hours",
  "24h": "Last 24 hours",
  "3d": "Last 3 days",
  "7d": "Last 7 days",
  "14d": "Last 14 days",
  "30d": "Last 30 days",
};

export type MetricUnit =
  | "requestsPerSecond"
  | "ratio"
  | "milliseconds"
  | "millicores"
  | "bytes"
  | "bytesPerSecond"
  | "count";

type UnitView = {
  format: (value: number) => ValueParts;
  tick: (value: number) => string;
  yFloor: number;
};

const decimal = new Intl.NumberFormat("en", { maximumFractionDigits: 2 });

function trimZeros(label: string): string {
  return label.replace(/(\.\d*?)0+(\D*)$/, "$1$2").replace(/\.(\D*)$/, "$1");
}

function percentOfRatio(ratio: number): string {
  const percent = ratio * 100;
  if (percent > 0 && percent < 0.01) {
    return "<0.01%";
  }
  return trimZeros(formatTooltipPercent(percent));
}

function latency(ms: number): string {
  return trimZeros(formatLatency(Math.round(ms)));
}

// The shared byte formatters ("1.5 MiB") overflow the 36px y-axis gutter.
const BYTE_SUFFIXES = ["B", "K", "M", "G", "T"];

function compactBytes(bytes: number): string {
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), BYTE_SUFFIXES.length - 1);
  const scaled = bytes / 1024 ** Math.max(exponent, 0);
  return `${Number(scaled.toPrecision(2))}${BYTE_SUFFIXES[Math.max(exponent, 0)]}`;
}

// formatCpuParts reads allocations (quarter vCPUs); usage is never quarter-aligned.
function formatMillicores(millicores: number): ValueParts {
  return { value: decimal.format(millicores / 1000), unit: "vCPU" };
}

function nonZero(format: (v: number) => string) {
  return (v: number) => (Number.isFinite(v) && v > 0 ? format(v) : "");
}

export const UNITS: Record<MetricUnit, UnitView> = {
  requestsPerSecond: {
    format: (v) => ({ value: decimal.format(v), unit: "req/s" }),
    tick: nonZero(formatCompactQuantity),
    yFloor: 1,
  },
  ratio: {
    format: (v) => ({ value: percentOfRatio(v) }),
    tick: nonZero(percentOfRatio),
    yFloor: 0.01,
  },
  milliseconds: {
    format: (v) => ({ value: latency(v) }),
    tick: nonZero((v) => (Math.round(v) >= 1000 ? latency(v) : String(Math.round(v)))),
    yFloor: 10,
  },
  millicores: {
    format: formatMillicores,
    tick: nonZero((v) => decimal.format(v / 1000)),
    yFloor: 10,
  },
  bytes: {
    format: formatNetworkBytesParts,
    tick: nonZero(compactBytes),
    yFloor: 1024 * 1024,
  },
  bytesPerSecond: {
    format: (v) => (v === 0 ? { value: "0", unit: "B/s" } : formatBytesPerSecondParts(v)),
    tick: nonZero((v) => `${compactBytes(v)}/s`),
    yFloor: 1024,
  },
  count: {
    format: (v) => ({ value: formatCompactQuantity(v) }),
    tick: nonZero(formatCompactQuantity),
    yFloor: 1,
  },
};

export function formatValue(unit: MetricUnit, value: number): string {
  return joinParts(UNITS[unit].format(value));
}

export type SeriesView = { label: string; color: string };

type MetricView<M extends MetricId> = {
  title: string;
  unit: MetricUnit;
  primary: SeriesKey<M>;
  series: Record<SeriesKey<M>, SeriesView>;
};

const ACTIVITY = "var(--color-activity)";

export function sumOf<P>(points: ReadonlyArray<P>, value: (p: P) => number | null): number {
  return points.reduce((acc, p) => acc + (value(p) ?? 0), 0);
}

export function errorRate(points: ReadonlyArray<TrafficPoint>, bucketSec: number): number | null {
  const total = sumOf(points, (p) => p.requests * bucketSec);
  return total > 0 ? sumOf(points, (p) => p.errors) / total : null;
}

export const METRIC_VIEWS: { [M in MetricId]: MetricView<M> } = {
  requests: {
    primary: "requests",
    title: "Requests",
    unit: "requestsPerSecond",
    series: { requests: { label: "Requests/s", color: ACTIVITY } },
  },
  errors: {
    primary: "errorRate",
    title: "Error rate",
    unit: "ratio",
    series: { errorRate: { label: "5xx rate", color: "var(--color-error-9)" } },
  },
  latency: {
    primary: "p95",
    title: "Latency",
    unit: "milliseconds",
    series: {
      p50: { label: "p50", color: "var(--color-info-7)" },
      p95: { label: "p95", color: "var(--color-info-9)" },
      p99: { label: "p99", color: "var(--color-info-11)" },
    },
  },
  cpu: {
    primary: "cpu",
    title: "CPU",
    unit: "millicores",
    series: { cpu: { label: "CPU", color: ACTIVITY } },
  },
  memory: {
    primary: "memory",
    title: "Memory",
    unit: "bytes",
    series: { memory: { label: "Memory", color: ACTIVITY } },
  },
  disk: {
    primary: "disk",
    title: "Disk",
    unit: "bytes",
    series: { disk: { label: "Disk", color: ACTIVITY } },
  },
  network: {
    primary: "ingress",
    title: "Network",
    unit: "bytesPerSecond",
    series: {
      ingress: { label: "In", color: ACTIVITY },
      egress: { label: "Out", color: "var(--color-feature-9)" },
    },
  },
  instances: {
    primary: "instances",
    title: "Instances",
    unit: "count",
    series: { instances: { label: "Instances", color: ACTIVITY } },
  },
};

export function primarySeries<M extends MetricId>(metric: M): SeriesView {
  const view: MetricView<M> = METRIC_VIEWS[metric];
  return view.series[view.primary];
}

type SeriesPoint = { t: number } & { [key: string]: number | null };

export type SourceOf<M extends MetricId> = (typeof METRIC_CATALOG)[M]["source"];

export function seriesChart<M extends MetricId>(
  points: ReadonlyArray<SourcePoint<SourceOf<M>>>,
  metric: M,
): AreaChartPoint[] {
  const rows: ReadonlyArray<SeriesPoint> = points;
  return rows.map((p) => {
    const point: AreaChartPoint = { originalTimestamp: p.t };
    for (const key of METRIC_CATALOG[metric].series) {
      point[key] = p[key] ?? undefined;
    }
    return point;
  });
}

export type ChartState =
  | { status: "loading" }
  | ({ status: "failed" } & Failure)
  | { status: "empty" }
  | { status: "ready"; grid: Grid; chart: AreaChartPoint[] };

export function chartStateOf(grid: Grid, chart: AreaChartPoint[]): ChartState {
  const reported = chart.some((point) =>
    Object.entries(point).some(
      ([key, value]) => key !== "originalTimestamp" && value !== undefined,
    ),
  );
  return reported ? { status: "ready", grid, chart } : { status: "empty" };
}

export function cardState<M extends MetricId>(sources: Sources, metric: M): ChartState {
  const source: SourceOf<M> = METRIC_CATALOG[metric].source;
  const loaded: Loadable<SourceResult<SourceOf<M>>> = sources[source];
  return loaded.status !== "ready"
    ? loaded
    : chartStateOf(loaded.data.grid, seriesChart(loaded.data.points, metric));
}
