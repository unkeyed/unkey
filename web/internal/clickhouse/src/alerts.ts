import { z } from "zod";
import {
  type SigmaAlertMetric,
  alertBaselineMinimums,
  alertFixedThresholds,
  alertMinimumStddevRatio,
  alertStddevFloors,
  alertThresholdSigma,
  requestDropActivityFloor,
  requestDropMinimumAbsoluteLoss,
  requestDropMinimumActiveBuckets,
  requestDropRecentLevelFraction,
} from "./alert-thresholds";
import type { Querier } from "./client";

const fiveMinutesMs = 5 * 60 * 1000;
const hourMs = 60 * 60 * 1000;
const baselineMs = 24 * hourMs;

export function alertSeriesBaselineStartMs(startMs: number, appCreatedAtMs: number): number {
  const alignedAppCreatedAtMs = Math.floor(appCreatedAtMs / fiveMinutesMs) * fiveMinutesMs;
  return Math.max(startMs - baselineMs, alignedAppCreatedAtMs);
}

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
    appCreatedAtMs: z.int().nonnegative(),
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

function denseObservedBuckets(query: string): string {
  return `
    SELECT time, toFloat64(value) AS value, toUInt8(bucket_observed) AS bucket_observed
    FROM (${query})
    ORDER BY time ASC
    WITH FILL
      FROM {baselineStartMs: Int64}
      TO {endMs: Int64}
      STEP {bucketMs: UInt32}`;
}

function frontlineQuery(metric: "error_5xx" | "error_4xx" | "requests"): {
  query: string;
  tableName: string;
} {
  const tableName = "frontline_requests_per_5m_v1";
  if (metric === "requests") {
    return {
      query: denseObservedBuckets(`
        SELECT
          toInt64(toUnixTimestamp(metric_source.time) * 1000) AS time,
          sum(count) AS value,
          toUInt8(1) AS bucket_observed
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
        toFloat64(requests) AS requests,
        toUInt8(bucket_observed) AS bucket_observed
      FROM (
        SELECT
          toInt64(toUnixTimestamp(metric_source.time) * 1000) AS time,
          sumIf(count, ${errorFilter}) AS errors,
          sum(count) AS requests,
          toUInt8(1) AS bucket_observed
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

function resourceCounterQuery(expression: string): { query: string; tableName: string } {
  const fiveMinuteQuery = `
    SELECT bucket AS time, sum(container_value) AS value, toUInt8(1) AS bucket_observed
    FROM (
      SELECT
        toInt64(toUnixTimestamp(toStartOfInterval(time, INTERVAL 5 MINUTE)) * 1000) AS bucket,
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
    GROUP BY bucket`;
  return {
    query: denseObservedBuckets(fiveMinuteQuery),
    tableName: "instance_resources_per_minute_v1",
  };
}

function memoryQuery(resolution: "5m" | "1h"): { query: string; tableName: string } {
  const fiveMinuteQuery = `
    SELECT time, ifNotFinite(avgIf(instance_value, instance_memory_valid), 0.) AS value
    FROM (
      SELECT
        time,
        instance_id,
        ifNotFinite(avgIf(container_value, container_memory_valid), 0.) AS instance_value,
        countIf(container_memory_valid) > 0 AS instance_memory_valid
      FROM (
        SELECT
          toInt64(toUnixTimestamp(metric_source.time) * 1000) AS time,
          metric_source.instance_id,
          metric_source.container_uid,
          if(
            sum(utilization_samples) = 0,
            0.,
            sum(utilization_sum) / sum(utilization_samples)
          ) AS container_value,
          sum(utilization_samples) > 0 AS container_memory_valid
        FROM {tableName: Identifier} AS metric_source
        PREWHERE metric_source.workspace_id = {workspaceId: String}
          AND metric_source.app_id = {appId: String}
          AND metric_source.environment_id = {environmentId: String}
          AND metric_source.time >= fromUnixTimestamp64Milli({baselineStartMs: Int64})
          AND metric_source.time < fromUnixTimestamp64Milli({endMs: Int64})
        GROUP BY time, metric_source.instance_id, metric_source.container_uid
      )
      GROUP BY time, instance_id
    )
    GROUP BY time`;
  const displayQuery =
    resolution === "5m"
      ? fiveMinuteQuery
      : `
        SELECT
          intDiv(five_minute.time, ${hourMs}) * ${hourMs} AS time,
          avg(five_minute.value) AS value
        FROM (${fiveMinuteQuery}) AS five_minute
        GROUP BY time`;
  return {
    query: denseBuckets(displayQuery),
    tableName: "instance_resources_container_per_5m_v1",
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

function withExpectedRange(observedQuery: string, metric: SigmaAlertMetric): string {
  const trailingWindow = "ORDER BY time ROWS BETWEEN 288 PRECEDING AND 1 PRECEDING";
  const effectiveStddev = `greatest(
    expected_stddev,
    ${alertMinimumStddevRatio} * expected_mean,
    ${alertStddevFloors[metric]}
  )`;
  const lowerBound =
    metric === "requests"
      ? `if(
          observed_baseline_buckets < ${alertBaselineMinimums.requests_drop}
            OR recent_active_buckets < ${requestDropMinimumActiveBuckets},
          CAST(NULL, 'Nullable(Float64)'),
          toNullable(greatest(
            0,
            least(
              recent_median * ${requestDropRecentLevelFraction},
              recent_median - ${requestDropMinimumAbsoluteLoss}
            )
          ))
        )`
      : "CAST(NULL, 'Nullable(Float64)')";
  const requestDropStats =
    metric === "requests"
      ? `quantileExactInclusive(0.5)(lifetime_value) OVER (
          ORDER BY time ROWS BETWEEN 12 PRECEDING AND 1 PRECEDING
        ) AS recent_median,
        countIf(lifetime_value >= ${requestDropActivityFloor}) OVER (
          ORDER BY time ROWS BETWEEN 12 PRECEDING AND 1 PRECEDING
        ) AS recent_active_buckets,`
      : "";

  return `
    SELECT
      time,
      value,
      toNullable(expected_mean) AS expectedMean,
      ${lowerBound} AS lowerBound,
      if(
        observed_baseline_buckets < ${alertBaselineMinimums[metric]},
        CAST(NULL, 'Nullable(Float64)'),
        toNullable(expected_mean + ${alertThresholdSigma} * ${effectiveStddev})
      ) AS upperBound,
      CAST(NULL, 'Nullable(Float64)') AS limit
    FROM (
      SELECT
        time,
        value,
        avg(lifetime_value) OVER (
          ${trailingWindow}
        ) AS expected_mean,
        stddevPop(lifetime_value) OVER (
          ${trailingWindow}
        ) AS expected_stddev,
        ${requestDropStats}
        sum(lifetime_observed) OVER (${trailingWindow}) AS observed_baseline_buckets
      FROM (
        SELECT
          time,
          value,
          toNullable(value) AS lifetime_value,
          toUInt64(bucket_observed) AS lifetime_observed
        FROM (${observedQuery})
      )
    )
    WHERE time >= {startMs: Int64} AND time < {endMs: Int64}
    ORDER BY time ASC`;
}

function withExpectedErrorRatioRange(
  observedQuery: string,
  metric: "error_5xx" | "error_4xx",
): string {
  const trailingWindow = "ORDER BY time ROWS BETWEEN 288 PRECEDING AND 1 PRECEDING";
  const effectiveStddev = `greatest(
    expected_stddev,
    ${alertMinimumStddevRatio} * expected_mean,
    ${alertStddevFloors[metric]}
  )`;
  return `
    SELECT
      time,
      value,
      errors,
      requests,
      toNullable(expected_mean) AS expectedMean,
      CAST(NULL, 'Nullable(Float64)') AS lowerBound,
      if(
        observed_baseline_buckets < ${alertBaselineMinimums[metric]},
        CAST(NULL, 'Nullable(Float64)'),
        toNullable(expected_mean + ${alertThresholdSigma} * ${effectiveStddev})
      ) AS upperBound,
      CAST(NULL, 'Nullable(Float64)') AS limit
    FROM (
      SELECT
        time,
        value,
        errors,
        requests,
        if(
          sum(lifetime_requests) OVER (${trailingWindow}) = 0,
          0,
          sum(lifetime_errors) OVER (${trailingWindow}) /
            sum(lifetime_requests) OVER (${trailingWindow})
        ) AS expected_mean,
        stddevPop(lifetime_value) OVER (${trailingWindow}) AS expected_stddev,
        sum(lifetime_observed) OVER (${trailingWindow}) AS observed_baseline_buckets
      FROM (
        SELECT
          time,
          value,
          errors,
          requests,
          if(requests > 0, toNullable(value), CAST(NULL, 'Nullable(Float64)')) AS lifetime_value,
          toNullable(errors) AS lifetime_errors,
          toNullable(requests) AS lifetime_requests,
          toUInt64(bucket_observed) AS lifetime_observed
        FROM (${observedQuery})
      )
    )
    WHERE time >= {startMs: Int64} AND time < {endMs: Int64}
    ORDER BY time ASC`;
}

function aggregateExpectedRangeToHours(
  fiveMinuteQuery: string,
  metric: SigmaAlertMetric | "error_5xx" | "error_4xx",
): string {
  const errorRatio = metric === "error_5xx" || metric === "error_4xx";
  const aggregate = errorRatio ? "avg" : "sum";
  const value = errorRatio
    ? "if(sum(requests) = 0, 0., sum(errors) / sum(requests))"
    : "sum(value)";
  const nullableAggregate = (field: "expectedMean" | "lowerBound" | "upperBound") => `if(
    count(${field}) < count(),
    CAST(NULL, 'Nullable(Float64)'),
    toNullable(${aggregate}(${field}))
  )`;
  return `
    SELECT
      intDiv(five_minute.time, ${hourMs}) * ${hourMs} AS time,
      ${value} AS value,
      ${nullableAggregate("expectedMean")} AS expectedMean,
      ${nullableAggregate("lowerBound")} AS lowerBound,
      ${nullableAggregate("upperBound")} AS upperBound,
      CAST(NULL, 'Nullable(Float64)') AS limit
    FROM (${fiveMinuteQuery}) AS five_minute
    GROUP BY time
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
    const displayBucketMs = args.resolution === "5m" ? fiveMinutesMs : hourMs;
    let observedQuery: string;
    let tableName: string | undefined;
    let sigmaMetric: SigmaAlertMetric | undefined;
    let errorRatioMetric: "error_5xx" | "error_4xx" | undefined;
    let fixedLimit: number | null = null;

    switch (args.metric) {
      case "error_5xx":
      case "error_4xx": {
        const source = frontlineQuery(args.metric);
        observedQuery = source.query;
        tableName = source.tableName;
        errorRatioMetric = args.metric;
        break;
      }
      case "requests": {
        const source = frontlineQuery(args.metric);
        observedQuery = source.query;
        tableName = source.tableName;
        sigmaMetric = "requests";
        break;
      }
      case "egress_bytes": {
        const source = resourceCounterQuery(
          "greatest(0, max(network_egress_public_bytes_max) - min(network_egress_public_bytes_min))",
        );
        observedQuery = source.query;
        tableName = source.tableName;
        sigmaMetric = args.metric;
        break;
      }
      case "cpu_seconds": {
        const source = resourceCounterQuery(
          "greatest(0, max(cpu_usage_usec_max) - min(cpu_usage_usec_min)) / 1000000",
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
        fixedLimit = alertFixedThresholds.memory_utilization;
        break;
      }
      case "health":
        observedQuery = healthQuery();
        fixedLimit = null;
        break;
      default:
        throw new Error(`Unsupported alert series metric: ${args.metric satisfies never}`);
    }

    let queryText: string;
    if (errorRatioMetric !== undefined) {
      const expectedRange = withExpectedErrorRatioRange(observedQuery, errorRatioMetric);
      queryText =
        args.resolution === "1h"
          ? aggregateExpectedRangeToHours(expectedRange, errorRatioMetric)
          : expectedRange;
    } else if (sigmaMetric !== undefined) {
      const expectedRange = withExpectedRange(observedQuery, sigmaMetric);
      queryText =
        args.resolution === "1h"
          ? aggregateExpectedRangeToHours(expectedRange, sigmaMetric)
          : expectedRange;
    } else {
      queryText = withFixedLimit(observedQuery, fixedLimit);
    }

    const hasExpectedRange = errorRatioMetric !== undefined || sigmaMetric !== undefined;
    const query = ch.query({
      query: queryText,
      params: queryParams,
      schema: alertSeriesPoint,
    });
    return query({
      workspaceId: args.workspaceId,
      appId: args.appId,
      environmentId: args.environmentId,
      startMs: args.startMs,
      endMs: args.endMs,
      baselineStartMs: hasExpectedRange
        ? alertSeriesBaselineStartMs(args.startMs, args.appCreatedAtMs)
        : args.startMs,
      bucketMs: hasExpectedRange ? fiveMinutesMs : displayBucketMs,
      tableName: tableName ?? "instance_events_raw_v1",
    });
  };
}
