import { z } from "zod";
import type { Querier } from "./client";

const TIMESERIES_WINDOW_HOURS = 6;
const TIMESERIES_INTERVAL_MINUTES = 15;
const TIMESERIES_INTERVAL_SECONDS = TIMESERIES_INTERVAL_MINUTES * 60;
const CURRENT_RPS_WINDOW_MINUTES = 15;
const CURRENT_RPS_WINDOW_MS = CURRENT_RPS_WINDOW_MINUTES * 60 * 1000;

const TABLE = "default.sentinel_requests_raw_v1";

const SQL = {
  deploymentFilter: `
    workspace_id = {workspaceId: String}
    AND project_id = {projectId: String}
    AND deployment_id = {deploymentId: String}
    AND environment_id = {environmentId: String}`,

  // time (ms) >= X hours ago (ms)
  recentHours: "time >= toUnixTimestamp(now() - INTERVAL {windowHours: UInt8} HOUR) * 1000",

  // time (ms) >= X minutes ago (ms)
  recentMinutes: "time >= toUnixTimestamp(now() - INTERVAL {windowMinutes: UInt16} MINUTE) * 1000",

  // Truncate timestamp to interval bucket
  timeBucket:
    "toStartOfInterval(toDateTime(time / 1000), INTERVAL {intervalMinutes: UInt8} MINUTE)",

  // Fill gaps in timeseries with zeros
  fillGaps: `
    WITH FILL
      FROM toStartOfInterval(now() - INTERVAL {windowHours: UInt8} HOUR, INTERVAL {intervalMinutes: UInt8} MINUTE)
      TO toStartOfInterval(now(), INTERVAL {intervalMinutes: UInt8} MINUTE)
      STEP INTERVAL {intervalMinutes: UInt8} MINUTE`,
} as const;

export const percentileSchema = z.enum(["p50", "p75", "p90", "p95", "p99"]).default("p50");

export const PERCENTILE_VALUES = {
  p50: 0.5,
  p75: 0.75,
  p90: 0.9,
  p95: 0.95,
  p99: 0.99,
} as const;

const baseDeploymentParams = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
});

const rpsResponseSchema = z.object({ avg_rps: z.number() });
const latencyResponseSchema = z.object({ latency: z.number() });
const timeseriesPointSchema = z.object({ x: z.number().int(), y: z.number() });

// ─────────────────────────────────────────────────────────────
// Logs
// ─────────────────────────────────────────────────────────────

export const sentinelLogsRequestSchema = baseDeploymentParams.extend({
  limit: z.number().int().positive().default(50),
});

export type SentinelLogsRequest = z.infer<typeof sentinelLogsRequestSchema>;

export const sentinelLogsResponseSchema = z.object({
  request_id: z.string(),
  time: z.number().int(),
  deployment_id: z.string(),
  region: z.string(),
  method: z.string(),
  path: z.string(),
  response_status: z.number().int(),
  total_latency: z.number().int(),
  instance_latency: z.number().int(),
  sentinel_latency: z.number().int(),
});

export type SentinelLogsResponse = z.infer<typeof sentinelLogsResponseSchema>;

export function getSentinelLogs(ch: Querier) {
  return async (args: SentinelLogsRequest) => {
    const query = ch.query({
      query: `
        SELECT
          request_id, time, deployment_id, region, method, path,
          response_status, total_latency, instance_latency, sentinel_latency
        FROM ${TABLE}
        WHERE ${SQL.deploymentFilter}
          AND ${SQL.recentHours}
        ORDER BY time DESC
        LIMIT {limit: Int}`,
      params: sentinelLogsRequestSchema.extend({ windowHours: z.number() }),
      schema: sentinelLogsResponseSchema,
    });

    return query({ ...args, windowHours: TIMESERIES_WINDOW_HOURS });
  };
}

// ─────────────────────────────────────────────────────────────
// Sentinel / Instance RPS
// ─────────────────────────────────────────────────────────────

export const sentinelRpsRequestSchema = baseDeploymentParams.extend({
  sentinelId: z.string(),
});

export function getSentinelRps(ch: Querier) {
  return async (args: z.infer<typeof sentinelRpsRequestSchema>) => {
    const query = ch.query({
      query: `
        SELECT round(COUNT(*) * 1000.0 / {windowMs: UInt64}, 2) as avg_rps
        FROM ${TABLE}
        WHERE ${SQL.deploymentFilter}
          AND sentinel_id = {sentinelId: String}
          AND ${SQL.recentMinutes}`,
      params: sentinelRpsRequestSchema.extend({
        windowMinutes: z.number(),
        windowMs: z.number(),
      }),
      schema: rpsResponseSchema,
    });

    return query({
      ...args,
      windowMinutes: CURRENT_RPS_WINDOW_MINUTES,
      windowMs: CURRENT_RPS_WINDOW_MS,
    });
  };
}

export const instanceRpsRequestSchema = baseDeploymentParams.extend({
  instanceId: z.string(),
});

export function getInstanceRps(ch: Querier) {
  return async (args: z.infer<typeof instanceRpsRequestSchema>) => {
    const query = ch.query({
      query: `
        -- count * 1000 / ms = requests per second
        SELECT round(COUNT(*) * 1000.0 / {windowMs: UInt64}, 2) as avg_rps
        FROM ${TABLE}
        WHERE ${SQL.deploymentFilter}
          AND instance_id = {instanceId: String}
          AND ${SQL.recentMinutes}`,
      params: instanceRpsRequestSchema.extend({
        windowMinutes: z.number(),
        windowMs: z.number(),
      }),
      schema: rpsResponseSchema,
    });

    return query({
      ...args,
      windowMinutes: CURRENT_RPS_WINDOW_MINUTES,
      windowMs: CURRENT_RPS_WINDOW_MS,
    });
  };
}

// ─────────────────────────────────────────────────────────────
// Deployment RPS
// ─────────────────────────────────────────────────────────────

export const deploymentRpsRequestSchema = baseDeploymentParams;
export type DeploymentRpsRequest = z.infer<typeof deploymentRpsRequestSchema>;

export function getDeploymentRps(ch: Querier) {
  return async (args: DeploymentRpsRequest) => {
    const windowMs = TIMESERIES_WINDOW_HOURS * 60 * 60 * 1000;

    const query = ch.query({
      query: `
        -- count * 1000 / ms = requests per second
        SELECT round(COUNT(*) * 1000.0 / {windowMs: UInt64}, 2) as avg_rps
        FROM ${TABLE}
        WHERE ${SQL.deploymentFilter}
          AND ${SQL.recentHours}`,
      params: baseDeploymentParams.extend({
        windowHours: z.number(),
        windowMs: z.number(),
      }),
      schema: rpsResponseSchema,
    });

    return query({ ...args, windowHours: TIMESERIES_WINDOW_HOURS, windowMs });
  };
}

// ─────────────────────────────────────────────────────────────
// Deployment RPS Timeseries
// ─────────────────────────────────────────────────────────────

export const deploymentRpsTimeseriesRequestSchema = baseDeploymentParams;
export type DeploymentRpsTimeseriesRequest = z.infer<typeof deploymentRpsTimeseriesRequestSchema>;

export function getDeploymentRpsTimeseries(ch: Querier) {
  return async (args: DeploymentRpsTimeseriesRequest) => {
    const query = ch.query({
      query: `
        SELECT
          toUnixTimestamp(bucket) * 1000 as x,
          round(COUNT(*) / {intervalSeconds: UInt32}, 2) as y
        FROM (
          SELECT ${SQL.timeBucket} as bucket
          FROM ${TABLE}
          WHERE ${SQL.deploymentFilter}
            AND ${SQL.recentHours}
        )
        GROUP BY bucket
        ORDER BY bucket ASC
        ${SQL.fillGaps}`,
      params: baseDeploymentParams.extend({
        windowHours: z.number(),
        intervalMinutes: z.number(),
        intervalSeconds: z.number(),
      }),
      schema: timeseriesPointSchema,
    });

    return query({
      ...args,
      windowHours: TIMESERIES_WINDOW_HOURS,
      intervalMinutes: TIMESERIES_INTERVAL_MINUTES,
      intervalSeconds: TIMESERIES_INTERVAL_SECONDS,
    });
  };
}

// ─────────────────────────────────────────────────────────────
// Deployment Latency
// ─────────────────────────────────────────────────────────────

export const deploymentLatencyRequestSchema = baseDeploymentParams.extend({
  percentile: percentileSchema,
});

export type DeploymentLatencyRequest = z.infer<typeof deploymentLatencyRequestSchema>;

export function getDeploymentLatency(ch: Querier) {
  return async (args: DeploymentLatencyRequest) => {
    const percentileValue = PERCENTILE_VALUES[args.percentile];

    const query = ch.query({
      query: `
        SELECT round(quantile({percentileValue: Float64})(total_latency), 2) as latency
        FROM ${TABLE}
        WHERE ${SQL.deploymentFilter}
          AND ${SQL.recentMinutes}`,
      params: deploymentLatencyRequestSchema.extend({
        windowMinutes: z.number(),
        percentileValue: z.number(),
      }),
      schema: latencyResponseSchema,
    });

    return query({
      ...args,
      windowMinutes: TIMESERIES_WINDOW_HOURS * 60,
      percentileValue,
    });
  };
}

// ─────────────────────────────────────────────────────────────
// Deployment Latency Timeseries
// ─────────────────────────────────────────────────────────────

export const deploymentLatencyTimeseriesRequestSchema = baseDeploymentParams.extend({
  percentile: percentileSchema,
});

export type DeploymentLatencyTimeseriesRequest = z.infer<
  typeof deploymentLatencyTimeseriesRequestSchema
>;

export function getDeploymentLatencyTimeseries(ch: Querier) {
  return async (args: DeploymentLatencyTimeseriesRequest) => {
    const percentileValue = PERCENTILE_VALUES[args.percentile];

    const query = ch.query({
      query: `
        SELECT
          toUnixTimestamp(bucket) * 1000 as x,
          round(quantile({percentileValue: Float64})(total_latency), 2) as y
        FROM (
          SELECT ${SQL.timeBucket} as bucket, total_latency
          FROM ${TABLE}
          WHERE ${SQL.deploymentFilter}
            AND ${SQL.recentHours}
        )
        GROUP BY bucket
        ORDER BY bucket ASC
        ${SQL.fillGaps}`,
      params: deploymentLatencyTimeseriesRequestSchema.extend({
        windowHours: z.number(),
        intervalMinutes: z.number(),
        percentileValue: z.number(),
      }),
      schema: timeseriesPointSchema,
    });

    return query({
      ...args,
      windowHours: TIMESERIES_WINDOW_HOURS,
      intervalMinutes: TIMESERIES_INTERVAL_MINUTES,
      percentileValue,
    });
  };
}
