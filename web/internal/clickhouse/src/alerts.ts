import { z } from "zod";
import {
  type SigmaAlertMetric,
  alertMinimumLifetimeBuckets,
  alertMinimumStddevRatio,
  alertStddevFloors,
  alertThresholdSigma,
  requestDropMedianFraction,
} from "./alert-thresholds";
import type { Querier } from "./client";

const fiveMinutesMs = 5 * 60 * 1000;
const hourMs = 60 * 60 * 1000;
const baselineMs = 24 * hourMs;

export const alertSeriesMetric = z.enum([
  "error_5xx",
  "error_4xx",
  "requests",
  "egress_bytes",
  "cpu_seconds",
  "memory_utilization",
  "health",
]);

export const alertSeriesParams = z
  .object({
    workspaceId: z.string(),
    appId: z.string(),
    environmentId: z.string(),
    metric: alertSeriesMetric,
    resolution: z.enum(["5m", "1h"]),
    startMs: z.int().nonnegative(),
    endMs: z.int().nonnegative(),
  })
  .refine(({ startMs, endMs }) => startMs < endMs, {
    message: "startMs must be before endMs",
  });

export type AlertSeriesParams = z.infer<typeof alertSeriesParams>;

export const alertSeriesPoint = z.object({
  time: z.int(),
  value: z.number(),
  expectedMean: z.number().nullable(),
  lowerBound: z.number().nullable(),
  upperBound: z.number().nullable(),
  limit: z.number().nullable(),
});

const queryParams = z.object({
  workspaceId: z.string(),
  appId: z.string(),
  environmentId: z.string(),
  startMs: z.int(),
  endMs: z.int(),
  baselineStartMs: z.int(),
  bucketMs: z.int().positive(),
  tableName: z.string().optional(),
});

function denseBuckets(query: string): string {
  return `
    SELECT time, toFloat64(value) AS value
    FROM (${query})
    ORDER BY time ASC
    WITH FILL
      FROM {baselineStartMs: Int64}
      TO {endMs: Int64}
      STEP {bucketMs: UInt32}`;
}

function frontlineQuery(
  metric: "error_5xx" | "error_4xx" | "requests",
  resolution: "5m" | "1h",
): { query: string; tableName: string } {
  const tableName =
    resolution === "5m" ? "frontline_requests_per_5m_v1" : "frontline_requests_per_hour_v1";
  if (metric === "requests") {
    return {
      query: denseBuckets(`
        SELECT
          toInt64(toUnixTimestamp(metric_source.time) * 1000) AS time,
          sum(count) AS value
        FROM {tableName: Identifier} AS metric_source
        PREWHERE metric_source.workspace_id = {workspaceId: String}
          AND metric_source.app_id = {appId: String}
          AND metric_source.environment_id = {environmentId: String}
          AND metric_source.time >= fromUnixTimestamp64Milli({baselineStartMs: Int64})
          AND metric_source.time < fromUnixTimestamp64Milli({endMs: Int64})
        GROUP BY time`),
      tableName,
    };
  }

  const errorFilter =
    metric === "error_5xx"
      ? "response_status >= 500 AND response_status < 600"
      : "response_status >= 400 AND response_status < 500";

  return {
    query: `
      SELECT
        time,
        if(requests = 0, 0, toFloat64(errors) / requests) AS value,
        toFloat64(errors) AS errors,
        toFloat64(requests) AS requests
      FROM (
        SELECT
          toInt64(toUnixTimestamp(metric_source.time) * 1000) AS time,
          sumIf(count, ${errorFilter}) AS errors,
          sum(count) AS requests
        FROM {tableName: Identifier} AS metric_source
        PREWHERE metric_source.workspace_id = {workspaceId: String}
          AND metric_source.app_id = {appId: String}
          AND metric_source.environment_id = {environmentId: String}
          AND metric_source.time >= fromUnixTimestamp64Milli({baselineStartMs: Int64})
          AND metric_source.time < fromUnixTimestamp64Milli({endMs: Int64})
        GROUP BY time
      )
      ORDER BY time ASC
      WITH FILL
        FROM {baselineStartMs: Int64}
        TO {endMs: Int64}
        STEP {bucketMs: UInt32}`,
    tableName,
  };
}

function resourceCounterQuery(
  expression: string,
  aggregate: string,
  resolution: "5m" | "1h",
): { query: string; tableName: string } {
  const bucket =
    resolution === "5m" ? "toStartOfInterval(time, INTERVAL 5 MINUTE)" : "toStartOfHour(time)";
  return {
    query: denseBuckets(`
      SELECT bucket AS time, ${aggregate} AS value
      FROM (
        SELECT
          toInt64(toUnixTimestamp(${bucket}) * 1000) AS bucket,
          container_uid,
          ${expression} AS container_value
        FROM {tableName: Identifier}
        PREWHERE workspace_id = {workspaceId: String}
          AND app_id = {appId: String}
          AND environment_id = {environmentId: String}
          AND resource_type = 'deployment'
          AND time >= fromUnixTimestamp64Milli({baselineStartMs: Int64})
          AND time < fromUnixTimestamp64Milli({endMs: Int64})
        GROUP BY bucket, container_uid
      )
      GROUP BY bucket`),
    tableName:
      resolution === "5m" ? "instance_resources_per_minute_v1" : "instance_resources_per_hour_v1",
  };
}

function memoryQuery(resolution: "5m" | "1h"): { query: string; tableName: string } {
  const bucket =
    resolution === "5m"
      ? "toStartOfInterval(metric_source.time, INTERVAL 5 MINUTE)"
      : "toStartOfHour(metric_source.time)";
  return {
    query: denseBuckets(`
      SELECT time, avg(instance_value) AS value
      FROM (
        SELECT
          toInt64(toUnixTimestamp(${bucket}) * 1000) AS time,
          metric_source.container_uid,
          if(max(memory_allocated_bytes_max) = 0, 0,
            max(memory_bytes_max) / max(memory_allocated_bytes_max)) AS instance_value
        FROM {tableName: Identifier} AS metric_source
        PREWHERE metric_source.workspace_id = {workspaceId: String}
          AND metric_source.app_id = {appId: String}
          AND metric_source.environment_id = {environmentId: String}
          AND metric_source.resource_type = 'deployment'
          AND metric_source.time >= fromUnixTimestamp64Milli({baselineStartMs: Int64})
          AND metric_source.time < fromUnixTimestamp64Milli({endMs: Int64})
        GROUP BY time, metric_source.container_uid
      )
      GROUP BY time`),
    tableName:
      resolution === "5m" ? "instance_resources_per_minute_v1" : "instance_resources_per_hour_v1",
  };
}

function healthQuery(): string {
  return denseBuckets(`
    SELECT
      intDiv(time, {bucketMs: UInt32}) * {bucketMs: UInt32} AS time,
      countIf(
        (event_kind = 'terminated' AND reason = 'OOMKilled')
        OR (event_kind = 'waiting' AND reason = 'CrashLoopBackOff')
      ) AS value
    FROM default.instance_events_raw_v1
    PREWHERE workspace_id = {workspaceId: String}
      AND app_id = {appId: String}
      AND environment_id = {environmentId: String}
      AND time >= {baselineStartMs: Int64}
      AND time < {endMs: Int64}
    GROUP BY time`);
}

function firstBucketQuery(metric: SigmaAlertMetric): string {
  const resourceFilter =
    metric === "egress_bytes" || metric === "cpu_seconds"
      ? "AND metric_lifetime.resource_type = 'deployment'"
      : "";
  return `
    SELECT coalesce(
      toInt64(toUnixTimestamp(minOrNull(metric_lifetime.time)) * 1000),
      {endMs: Int64}
    )
    FROM {tableName: Identifier} AS metric_lifetime
    PREWHERE metric_lifetime.workspace_id = {workspaceId: String}
      AND metric_lifetime.app_id = {appId: String}
      AND metric_lifetime.environment_id = {environmentId: String}
      ${resourceFilter}
      AND metric_lifetime.time < fromUnixTimestamp64Milli({endMs: Int64})`;
}

function withExpectedRange(
  observedQuery: string,
  metric: SigmaAlertMetric,
  windowBuckets: number,
  includeRequestDropThreshold: boolean,
): string {
  const effectiveStddev = `greatest(
    expected_stddev,
    ${alertMinimumStddevRatio} * expected_mean,
    ${alertStddevFloors[metric]}
  )`;
  const lowerBound =
    metric === "requests" && includeRequestDropThreshold
      ? `if(
          lifetime_buckets < ${alertMinimumLifetimeBuckets},
          CAST(NULL, 'Nullable(Float64)'),
          toNullable(greatest(0, recent_median * ${requestDropMedianFraction}))
        )`
      : "CAST(NULL, 'Nullable(Float64)')";
  const recentMedian =
    metric === "requests" && includeRequestDropThreshold
      ? `quantileExact(0.5)(lifetime_value) OVER (
          ORDER BY time ROWS BETWEEN 12 PRECEDING AND 1 PRECEDING
        ) AS recent_median,`
      : "";

  return `
    WITH (${firstBucketQuery(metric)}) AS first_bucket_time
    SELECT
      time,
      value,
      toNullable(expected_mean) AS expectedMean,
      ${lowerBound} AS lowerBound,
      if(
        lifetime_buckets < ${alertMinimumLifetimeBuckets},
        CAST(NULL, 'Nullable(Float64)'),
        toNullable(expected_mean + ${alertThresholdSigma} * ${effectiveStddev})
      ) AS upperBound,
      CAST(NULL, 'Nullable(Float64)') AS limit
    FROM (
      SELECT
        time,
        value,
        avg(lifetime_value) OVER (
          ORDER BY time ROWS BETWEEN ${windowBuckets} PRECEDING AND 1 PRECEDING
        ) AS expected_mean,
        stddevPop(lifetime_value) OVER (
          ORDER BY time ROWS BETWEEN ${windowBuckets} PRECEDING AND 1 PRECEDING
        ) AS expected_stddev,
        ${recentMedian}
        count(lifetime_value) OVER (
          ORDER BY time ROWS BETWEEN ${windowBuckets} PRECEDING AND 1 PRECEDING
        ) AS lifetime_buckets
      FROM (
        SELECT
          time,
          value,
          if(
            time >= first_bucket_time,
            toNullable(value),
            CAST(NULL, 'Nullable(Float64)')
          ) AS lifetime_value
        FROM (${observedQuery})
      )
    )
    WHERE time >= {startMs: Int64} AND time < {endMs: Int64}
    ORDER BY time ASC`;
}

function withExpectedErrorRatioRange(
  observedQuery: string,
  metric: "error_5xx" | "error_4xx",
  windowBuckets: number,
): string {
  const trailingWindow = `ORDER BY time ROWS BETWEEN ${windowBuckets} PRECEDING AND 1 PRECEDING`;
  const effectiveStddev = `greatest(
    expected_stddev,
    ${alertMinimumStddevRatio} * expected_mean,
    ${alertStddevFloors[metric]}
  )`;
  return `
    WITH (${firstBucketQuery(metric)}) AS first_bucket_time
    SELECT
      time,
      value,
      toNullable(expected_mean) AS expectedMean,
      CAST(NULL, 'Nullable(Float64)') AS lowerBound,
      if(
        lifetime_buckets < ${alertMinimumLifetimeBuckets},
        CAST(NULL, 'Nullable(Float64)'),
        toNullable(expected_mean + ${alertThresholdSigma} * ${effectiveStddev})
      ) AS upperBound,
      CAST(NULL, 'Nullable(Float64)') AS limit
    FROM (
      SELECT
        time,
        value,
        if(
          sum(lifetime_requests) OVER (${trailingWindow}) = 0,
          0,
          sum(lifetime_errors) OVER (${trailingWindow}) /
            sum(lifetime_requests) OVER (${trailingWindow})
        ) AS expected_mean,
        stddevPop(lifetime_value) OVER (${trailingWindow}) AS expected_stddev,
        count(lifetime_value) OVER (${trailingWindow}) AS lifetime_buckets
      FROM (
        SELECT
          time,
          value,
          if(time >= first_bucket_time, toNullable(value), CAST(NULL, 'Nullable(Float64)')) AS lifetime_value,
          if(time >= first_bucket_time, toNullable(errors), CAST(NULL, 'Nullable(Float64)')) AS lifetime_errors,
          if(time >= first_bucket_time, toNullable(requests), CAST(NULL, 'Nullable(Float64)')) AS lifetime_requests
        FROM (${observedQuery})
      )
    )
    WHERE time >= {startMs: Int64} AND time < {endMs: Int64}
    ORDER BY time ASC`;
}

function withFixedLimit(observedQuery: string, limit: number | null): string {
  const limitExpression =
    limit === null ? "CAST(NULL, 'Nullable(Float64)')" : `toNullable(${limit})`;
  return `
    SELECT
      time,
      value,
      CAST(NULL, 'Nullable(Float64)') AS expectedMean,
      CAST(NULL, 'Nullable(Float64)') AS lowerBound,
      CAST(NULL, 'Nullable(Float64)') AS upperBound,
      ${limitExpression} AS limit
    FROM (${observedQuery})
    WHERE time >= {startMs: Int64} AND time < {endMs: Int64}
    ORDER BY time ASC`;
}

export function getAlertSeries(ch: Querier) {
  return (args: AlertSeriesParams) => {
    const bucketMs = args.resolution === "5m" ? fiveMinutesMs : hourMs;
    const windowBuckets = baselineMs / bucketMs;
    let observedQuery: string;
    let tableName: string | undefined;
    let sigmaMetric: SigmaAlertMetric | undefined;
    let errorRatioMetric: "error_5xx" | "error_4xx" | undefined;
    let fixedLimit: number | null = null;

    switch (args.metric) {
      case "error_5xx":
      case "error_4xx": {
        const source = frontlineQuery(args.metric, args.resolution);
        observedQuery = source.query;
        tableName = source.tableName;
        errorRatioMetric = args.metric;
        break;
      }
      case "requests": {
        const source = frontlineQuery(args.metric, args.resolution);
        observedQuery = source.query;
        tableName = source.tableName;
        sigmaMetric = "requests";
        break;
      }
      case "egress_bytes": {
        const source = resourceCounterQuery(
          "max(network_egress_public_bytes_max) - min(network_egress_public_bytes_min)",
          "sum(container_value)",
          args.resolution,
        );
        observedQuery = source.query;
        tableName = source.tableName;
        sigmaMetric = args.metric;
        break;
      }
      case "cpu_seconds": {
        const source = resourceCounterQuery(
          "max(cpu_usage_usec_max) - min(cpu_usage_usec_min)",
          "sum(container_value) / 1000000",
          args.resolution,
        );
        observedQuery = source.query;
        tableName = source.tableName;
        sigmaMetric = args.metric;
        break;
      }
      case "memory_utilization": {
        const source = memoryQuery(args.resolution);
        observedQuery = source.query;
        tableName = source.tableName;
        fixedLimit = 0.9;
        break;
      }
      case "health":
        observedQuery = healthQuery();
        fixedLimit = null;
        break;
      default:
        throw new Error(`Unsupported alert series metric: ${args.metric satisfies never}`);
    }

    const query = ch.query({
      query:
        errorRatioMetric !== undefined
          ? withExpectedErrorRatioRange(observedQuery, errorRatioMetric, windowBuckets)
          : sigmaMetric !== undefined
            ? withExpectedRange(observedQuery, sigmaMetric, windowBuckets, args.resolution === "5m")
            : withFixedLimit(observedQuery, fixedLimit),
      params: queryParams,
      schema: alertSeriesPoint,
    });
    return query({
      workspaceId: args.workspaceId,
      appId: args.appId,
      environmentId: args.environmentId,
      startMs: args.startMs,
      endMs: args.endMs,
      baselineStartMs: args.startMs - baselineMs,
      bucketMs,
      tableName: tableName ?? "instance_events_raw_v1",
    });
  };
}
