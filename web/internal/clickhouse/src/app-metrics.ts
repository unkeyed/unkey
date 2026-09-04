import { z } from "zod";
import type { Querier } from "./client";

// App-scoped metrics for the app Metrics page: every instance of one app in
// one environment, bucketed over a fixed window and optionally split by
// instance or by deployment. Deployment-scoped charts live in resources.ts and
// frontline.ts; this module differs in scope (app + environment) and in the
// split dimension, which is what lets a spike be traced to the deploy that
// caused it.

export const APP_METRICS_WINDOWS = ["1h", "6h", "1d", "1w", "30d"] as const;
export type AppMetricsWindow = (typeof APP_METRICS_WINDOWS)[number];

export const APP_METRICS_GROUPS = ["none", "instance", "deployment"] as const;
export type AppMetricsGroup = (typeof APP_METRICS_GROUPS)[number];

export const APP_RESOURCE_METRICS = ["cpu", "memory", "disk", "egress", "ingress"] as const;
export type AppResourceMetric = (typeof APP_RESOURCE_METRICS)[number];

type WindowSpec = {
  windowSeconds: number;
  bucketSeconds: number;
  resourceTable: string;
  requestTable: string;
};

// Bucket sizes target 60-300 points per chart. Rollups coarser than the
// bucket are never read; finer rollups are re-bucketed in SQL, which is safe
// because every column is a min/max/sum over a container's monotone counters.
export const APP_METRICS_WINDOW_CONFIG: Record<AppMetricsWindow, WindowSpec> = {
  "1h": {
    windowSeconds: 60 * 60,
    bucketSeconds: 60,
    resourceTable: "instance_resources_per_minute_v1",
    requestTable: "frontline_requests_per_minute_v1",
  },
  "6h": {
    windowSeconds: 6 * 60 * 60,
    bucketSeconds: 5 * 60,
    resourceTable: "instance_resources_per_minute_v1",
    requestTable: "frontline_requests_per_5m_v1",
  },
  "1d": {
    windowSeconds: 24 * 60 * 60,
    bucketSeconds: 15 * 60,
    resourceTable: "instance_resources_per_minute_v1",
    requestTable: "frontline_requests_per_15m_v1",
  },
  "1w": {
    windowSeconds: 7 * 24 * 60 * 60,
    bucketSeconds: 60 * 60,
    resourceTable: "instance_resources_per_hour_v1",
    requestTable: "frontline_requests_per_hour_v1",
  },
  "30d": {
    windowSeconds: 30 * 24 * 60 * 60,
    bucketSeconds: 6 * 60 * 60,
    resourceTable: "instance_resources_per_hour_v1",
    requestTable: "frontline_requests_per_hour_v1",
  },
};

export type AppMetricsRange = {
  startMs: number;
  endMs: number;
  bucketSeconds: number;
};

export function resolveAppMetricsRange(window: AppMetricsWindow, nowMs: number): AppMetricsRange {
  const spec = APP_METRICS_WINDOW_CONFIG[window];
  const bucketMs = spec.bucketSeconds * 1000;
  const endMs = Math.floor(nowMs / bucketMs) * bucketMs + bucketMs;
  return { startMs: endMs - spec.windowSeconds * 1000, endMs, bucketSeconds: spec.bucketSeconds };
}

const scopeParams = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  appId: z.string(),
  environmentId: z.string(),
  window: z.enum(APP_METRICS_WINDOWS),
  groupBy: z.enum(APP_METRICS_GROUPS),
  startMs: z.number().int(),
  endMs: z.number().int(),
});

const internalParams = scopeParams.extend({
  tableName: z.string(),
  bucketSeconds: z.number().int(),
});

const seriesPointSchema = z.object({
  x: z.number().int(),
  series: z.string(),
  y: z.number(),
});
export type AppMetricsSeriesPoint = z.infer<typeof seriesPointSchema>;

const BUCKET =
  "toInt64(toUnixTimestamp(toStartOfInterval(time, INTERVAL {bucketSeconds: UInt32} SECOND)) * 1000)";

const SCOPE_FILTER = `
  workspace_id = {workspaceId: String}
  AND project_id = {projectId: String}
  AND app_id = {appId: String}
  AND environment_id = {environmentId: String}
  AND time >= fromUnixTimestamp64Milli({startMs: Int64})
  AND time <  fromUnixTimestamp64Milli({endMs: Int64})`;

// Per-container reduction inside a bucket. cpu and network are monotone
// counters, so the delta of the rollup's max and min is the amount consumed
// in that bucket. memory and disk are gauges, so the peak is kept.
const RESOURCE_CONTAINER_AGG: Record<AppResourceMetric, string> = {
  cpu: "sum(cpu_usage_usec_max - cpu_usage_usec_min)",
  memory: "max(memory_bytes_max)",
  disk: "max(disk_used_bytes_max)",
  egress: "sum(network_egress_public_bytes_max - network_egress_public_bytes_min)",
  ingress: "sum(network_ingress_public_bytes_max - network_ingress_public_bytes_min)",
};

const RESOURCE_SERIES: Record<AppMetricsGroup, string> = {
  none: "''",
  instance: "instance_id",
  deployment: "resource_id",
};

// Resource series per bucket. cpu is in microseconds consumed per bucket,
// egress/ingress in bytes per bucket, memory/disk in bytes at peak. Rates are
// derived by the caller from bucketSeconds so one query serves both the
// "rate" and the "total per bucket" presentations.
export function getAppResourceTimeseries(ch: Querier) {
  return async (args: z.infer<typeof scopeParams> & { metric: AppResourceMetric }) => {
    const spec = APP_METRICS_WINDOW_CONFIG[args.window];
    const query = ch.query({
      query: `
        SELECT x, series, toFloat64(sum(container_value)) AS y
        FROM (
          SELECT
            ${BUCKET} AS x,
            ${RESOURCE_SERIES[args.groupBy]} AS series,
            container_uid,
            ${RESOURCE_CONTAINER_AGG[args.metric]} AS container_value
          FROM {tableName: Identifier}
          WHERE ${SCOPE_FILTER}
            AND resource_type = 'deployment'
          GROUP BY x, series, container_uid
        )
        GROUP BY x, series
        ORDER BY x ASC, series ASC`,
      params: internalParams,
      schema: seriesPointSchema,
    });
    const { metric: _metric, ...rest } = args;
    return query({ ...rest, tableName: spec.resourceTable, bucketSeconds: spec.bucketSeconds });
  };
}

const STATUS_CLASS = "concat(toString(intDiv(response_status, 100)), 'xx')";

const REQUEST_SERIES: Record<AppMetricsGroup, string> = {
  none: STATUS_CLASS,
  instance: STATUS_CLASS,
  deployment: `concat(deployment_id, ':', ${STATUS_CLASS})`,
};

// Request counts per bucket, split by status class (2xx, 3xx, 4xx, 5xx). The
// deployment group keeps the status class as a suffix ("d_123:5xx") so the
// caller can show both total and error traffic per deployment from one read.
// The frontline rollups have no instance dimension, so the instance group
// falls back to status class.
export function getAppRequestTimeseries(ch: Querier) {
  return async (args: z.infer<typeof scopeParams>) => {
    const spec = APP_METRICS_WINDOW_CONFIG[args.window];
    const query = ch.query({
      query: `
        SELECT
          ${BUCKET} AS x,
          ${REQUEST_SERIES[args.groupBy]} AS series,
          toFloat64(sum(count)) AS y
        FROM {tableName: Identifier}
        WHERE ${SCOPE_FILTER}
        GROUP BY x, series
        ORDER BY x ASC, series ASC`,
      params: internalParams,
      schema: seriesPointSchema,
    });
    return query({ ...args, tableName: spec.requestTable, bucketSeconds: spec.bucketSeconds });
  };
}

const latencyPointSchema = z.object({
  x: z.number().int(),
  series: z.string(),
  p50: z.number(),
  p95: z.number(),
  p99: z.number(),
});
export type AppLatencyPoint = z.infer<typeof latencyPointSchema>;

const LATENCY_SERIES: Record<AppMetricsGroup, string> = {
  none: "''",
  instance: "''",
  deployment: "deployment_id",
};

// Latency percentiles per bucket in milliseconds. Buckets with no traffic
// come back as NaN from quantile merge and are coerced to zero so the chart
// can draw a gap-free line.
export function getAppLatencyTimeseries(ch: Querier) {
  return async (args: z.infer<typeof scopeParams>) => {
    const spec = APP_METRICS_WINDOW_CONFIG[args.window];
    const query = ch.query({
      query: `
        SELECT
          ${BUCKET} AS x,
          ${LATENCY_SERIES[args.groupBy]} AS series,
          ifNotFinite(round(quantileTDigestMerge(0.5)(latency_p50), 2), 0) AS p50,
          ifNotFinite(round(quantileTDigestMerge(0.95)(latency_p95), 2), 0) AS p95,
          ifNotFinite(round(quantileTDigestMerge(0.99)(latency_p99), 2), 0) AS p99
        FROM {tableName: Identifier}
        WHERE ${SCOPE_FILTER}
        GROUP BY x, series
        ORDER BY x ASC, series ASC`,
      params: internalParams,
      schema: latencyPointSchema,
    });
    return query({ ...args, tableName: spec.requestTable, bucketSeconds: spec.bucketSeconds });
  };
}
