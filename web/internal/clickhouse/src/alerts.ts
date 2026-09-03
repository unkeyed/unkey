import { z } from "zod";
import type { Querier } from "./client";

const bucketMs = 5 * 60 * 1000;

export const alertMetric = z.enum([
  "error_5xx",
  "error_4xx",
  "requests",
  "requests_drop",
  "egress_bytes",
  "cpu_seconds",
  "memory_utilization",
  "oom_killed",
  "crash_loop",
]);

const timeseriesScope = z.object({
  workspaceId: z.string(),
  appId: z.string(),
  environmentId: z.string(),
  startMs: z.int(),
  endMs: z.int(),
});

const queryParams = timeseriesScope.extend({
  bucketMs: z.int().positive(),
});

export const alertTimeseriesParams = timeseriesScope
  .extend({ metric: alertMetric })
  .refine(({ startMs, endMs }) => startMs <= endMs, { message: "startMs must not exceed endMs" });

export type AlertTimeseriesParams = z.infer<typeof alertTimeseriesParams>;

export const alertTimeseriesPoint = z.object({
  time: z.int(),
  value: z.number(),
});

function fillBuckets(query: string): string {
  return `
    SELECT time, toFloat64(value) AS value
    FROM (${query})
    ORDER BY time ASC
    WITH FILL
      FROM intDiv({startMs: Int64}, {bucketMs: UInt32}) * {bucketMs: UInt32}
      TO intDiv({endMs: Int64}, {bucketMs: UInt32}) * {bucketMs: UInt32} + {bucketMs: UInt32}
      STEP {bucketMs: UInt32}`;
}

function frontlineQuery(metric: "error_5xx" | "error_4xx" | "requests" | "requests_drop"): string {
  const expression = {
    error_5xx: "sumIf(count, response_status >= 500 AND response_status < 600)",
    error_4xx: "sumIf(count, response_status >= 400 AND response_status < 500)",
    requests: "sum(count)",
    requests_drop: "sum(count)",
  }[metric];

  return fillBuckets(`
    SELECT
      toInt64(toUnixTimestamp(time) * 1000) AS time,
      ${expression} AS value
    FROM default.frontline_requests_per_5m_v1
    PREWHERE workspace_id = {workspaceId: String}
      AND app_id = {appId: String}
      AND environment_id = {environmentId: String}
      AND time >= fromUnixTimestamp64Milli({startMs: Int64})
      AND time <= fromUnixTimestamp64Milli({endMs: Int64})
    GROUP BY time`);
}

function resourceCounterQuery(expression: string, aggregate: string): string {
  return fillBuckets(`
    SELECT
      toInt64(toUnixTimestamp(bucket) * 1000) AS time,
      ${aggregate} AS value
    FROM (
      SELECT
        toStartOfInterval(time, INTERVAL 5 MINUTE) AS bucket,
        container_uid,
        ${expression} AS container_value
      FROM default.instance_resources_per_minute_v1
      PREWHERE workspace_id = {workspaceId: String}
        AND app_id = {appId: String}
        AND environment_id = {environmentId: String}
        AND resource_type = 'deployment'
        AND time >= fromUnixTimestamp64Milli({startMs: Int64})
        AND time <= fromUnixTimestamp64Milli({endMs: Int64})
      GROUP BY bucket, container_uid
    )
    GROUP BY bucket`);
}

function memoryQuery(): string {
  return fillBuckets(`
    SELECT
      toInt64(toUnixTimestamp(toStartOfInterval(time, INTERVAL 5 MINUTE)) * 1000) AS time,
      if(
        max(memory_allocated_bytes_max) = 0,
        0,
        max(memory_bytes_max) / max(memory_allocated_bytes_max)
      ) AS value
    FROM default.instance_resources_per_minute_v1
    PREWHERE workspace_id = {workspaceId: String}
      AND app_id = {appId: String}
      AND environment_id = {environmentId: String}
      AND resource_type = 'deployment'
      AND time >= fromUnixTimestamp64Milli({startMs: Int64})
      AND time <= fromUnixTimestamp64Milli({endMs: Int64})
    GROUP BY toStartOfInterval(time, INTERVAL 5 MINUTE)`);
}

function instanceEventQuery(metric: "oom_killed" | "crash_loop"): string {
  const predicate =
    metric === "oom_killed"
      ? "event_kind = 'terminated' AND reason = 'OOMKilled'"
      : "event_kind = 'waiting' AND reason = 'CrashLoopBackOff'";

  return fillBuckets(`
    SELECT
      intDiv(time, {bucketMs: UInt32}) * {bucketMs: UInt32} AS time,
      countIf(${predicate}) AS value
    FROM default.instance_events_raw_v1
    PREWHERE workspace_id = {workspaceId: String}
      AND app_id = {appId: String}
      AND environment_id = {environmentId: String}
      AND time >= {startMs: Int64}
      AND time <= {endMs: Int64}
    GROUP BY time`);
}

export function getAlertTimeseries(ch: Querier) {
  return (args: AlertTimeseriesParams) => {
    let queryText: string;
    switch (args.metric) {
      case "error_5xx":
      case "error_4xx":
      case "requests":
      case "requests_drop":
        queryText = frontlineQuery(args.metric);
        break;
      case "egress_bytes":
        queryText = resourceCounterQuery(
          "max(network_egress_public_bytes_max) - min(network_egress_public_bytes_min)",
          "sum(container_value)",
        );
        break;
      case "cpu_seconds":
        queryText = resourceCounterQuery(
          "max(cpu_usage_usec_max) - min(cpu_usage_usec_min)",
          "sum(container_value) / 1000000",
        );
        break;
      case "memory_utilization":
        queryText = memoryQuery();
        break;
      case "oom_killed":
      case "crash_loop":
        queryText = instanceEventQuery(args.metric);
        break;
      default:
        throw new Error(`Unsupported alert metric: ${args.metric satisfies never}`);
    }

    const query = ch.query({
      query: queryText,
      params: queryParams,
      schema: alertTimeseriesPoint,
    });
    return query({
      workspaceId: args.workspaceId,
      appId: args.appId,
      environmentId: args.environmentId,
      startMs: args.startMs,
      endMs: args.endMs,
      bucketMs,
    });
  };
}
