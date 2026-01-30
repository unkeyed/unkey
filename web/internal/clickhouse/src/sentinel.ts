import { z } from "zod";
import type { Querier } from "./client";

const TIMESERIES_WINDOW_HOURS = 6;
const TIMESERIES_INTERVAL_MINUTES = 15;
const TIMESERIES_INTERVAL_SECONDS = TIMESERIES_INTERVAL_MINUTES * 60;
const CURRENT_RPS_WINDOW_MINUTES = 15;
const CURRENT_RPS_WINDOW_MS = CURRENT_RPS_WINDOW_MINUTES * 60 * 1000;

export const sentinelLogsRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
  limit: z.number().int().positive().default(50),
});

export type SentinelLogsRequestSchema = z.infer<typeof sentinelLogsRequestSchema>;

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

export type SentinelLogsResponseSchema = z.infer<typeof sentinelLogsResponseSchema>;

export function getSentinelLogs(ch: Querier) {
  return async (args: SentinelLogsRequestSchema) => {
    const query = ch.query({
      query: `
        SELECT
          request_id,
          time,
          deployment_id,
          region,
          method,
          path,
          response_status,
          total_latency,
          instance_latency,
          sentinel_latency
        FROM default.sentinel_requests_raw_v1
        WHERE workspace_id = {workspaceId: String}
          AND project_id = {projectId: String}
          AND environment_id = {environmentId: String}
          AND deployment_id = {deploymentId: String}
          AND time >= toUnixTimestamp(now() - INTERVAL {windowHours: UInt8} HOUR) * 1000
        ORDER BY time DESC
        LIMIT {limit: Int}`,
      params: sentinelLogsRequestSchema.extend({
        windowHours: z.number(),
      }),
      schema: sentinelLogsResponseSchema,
    });

    return query({
      ...args,
      windowHours: TIMESERIES_WINDOW_HOURS,
    });
  };
}

export const sentinelRpsRequestSchema = z.object({
  workspaceId: z.string(),
  deploymentId: z.string(),
  sentinelId: z.string(),
});

export const sentinelRpsResponseSchema = z.object({
  avg_rps: z.number(),
});

export const instanceRpsRequestSchema = z.object({
  workspaceId: z.string(),
  deploymentId: z.string(),
  instanceId: z.string(),
});

export const instanceRpsResponseSchema = z.object({
  avg_rps: z.number(),
});

function createRpsQuery<T extends z.ZodObject<z.ZodRawShape>>(
  ch: Querier,
  requestSchema: T,
  whereFields: Record<string, string>,
) {
  return async (args: z.infer<T>) => {
    const whereClauses = Object.entries(whereFields)
      .map(([field, type]) => `${field} = {${toCamelCase(field)}: ${type}}`)
      .join("\n          AND ");

    const query = ch.query({
      query: `
        SELECT
          round(COUNT(*) * 1000.0 / {windowMs: UInt64}, 2) as avg_rps
        FROM default.sentinel_requests_raw_v1
        WHERE ${whereClauses}
          AND time >= toUnixTimestamp(now() - INTERVAL {windowMinutes: UInt8} MINUTE) * 1000`,
      params: requestSchema.extend({
        windowMinutes: z.number(),
        windowMs: z.number(),
      }),
      schema: sentinelRpsResponseSchema,
    });

    return query({
      ...args,
      windowMinutes: CURRENT_RPS_WINDOW_MINUTES,
      windowMs: CURRENT_RPS_WINDOW_MS,
    });
  };
}

function toCamelCase(str: string): string {
  return str.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
}

export function getSentinelRps(ch: Querier) {
  return createRpsQuery(ch, sentinelRpsRequestSchema, {
    workspace_id: "String",
    deployment_id: "String",
    sentinel_id: "String",
  });
}

export function getInstanceRps(ch: Querier) {
  return createRpsQuery(ch, instanceRpsRequestSchema, {
    workspace_id: "String",
    deployment_id: "String",
    instance_id: "String",
  });
}

export const deploymentRpsRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
});

export const deploymentRpsResponseSchema = z.object({
  avg_rps: z.number(),
});

export function getDeploymentRps(ch: Querier) {
  return async (args: z.infer<typeof deploymentRpsRequestSchema>) => {
    const windowHours = TIMESERIES_WINDOW_HOURS;
    const windowMs = windowHours * 60 * 60 * 1000;

    const query = ch.query({
      query: `
        SELECT
          round(COUNT(*) * 1000.0 / {windowMs: UInt64}, 2) as avg_rps
        FROM default.sentinel_requests_raw_v1
        WHERE workspace_id = {workspaceId: String}
          AND project_id = {projectId: String}
          AND deployment_id = {deploymentId: String}
          AND environment_id = {environmentId: String}
          AND time >= toUnixTimestamp(now() - INTERVAL {windowHours: UInt8} HOUR) * 1000`,
      params: deploymentRpsRequestSchema.extend({
        windowHours: z.number(),
        windowMs: z.number(),
      }),
      schema: deploymentRpsResponseSchema,
    });

    return query({
      ...args,
      windowHours,
      windowMs,
    });
  };
}

export const deploymentRpsTimeseriesRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
});

export const deploymentRpsTimeseriesResponseSchema = z.object({
  x: z.number().int(),
  y: z.number(),
});

export function getDeploymentRpsTimeseries(ch: Querier) {
  return async (args: z.infer<typeof deploymentRpsTimeseriesRequestSchema>) => {
    const query = ch.query({
      query: `
        SELECT
          toUnixTimestamp(bucket) * 1000 as x,
          round(COUNT(*) / {intervalSeconds: UInt32}, 2) as y
        FROM (
          SELECT
            toStartOfInterval(toDateTime(time / 1000), INTERVAL {intervalMinutes: UInt8} MINUTE) as bucket
          FROM default.sentinel_requests_raw_v1
          WHERE workspace_id = {workspaceId: String}
            AND project_id = {projectId: String}
            AND deployment_id = {deploymentId: String}
            AND environment_id = {environmentId: String}
            AND time >= toUnixTimestamp(now() - INTERVAL {windowHours: UInt8} HOUR) * 1000
        )
        GROUP BY bucket
        ORDER BY bucket ASC
        WITH FILL
          FROM toStartOfInterval(now() - INTERVAL {windowHours: UInt8} HOUR, INTERVAL {intervalMinutes: UInt8} MINUTE)
          TO toStartOfInterval(now(), INTERVAL {intervalMinutes: UInt8} MINUTE)
          STEP INTERVAL {intervalMinutes: UInt8} MINUTE`,
      params: deploymentRpsTimeseriesRequestSchema.extend({
        windowHours: z.number(),
        intervalMinutes: z.number(),
        intervalSeconds: z.number(),
      }),
      schema: deploymentRpsTimeseriesResponseSchema,
    });

    return query({
      ...args,
      windowHours: TIMESERIES_WINDOW_HOURS,
      intervalMinutes: TIMESERIES_INTERVAL_MINUTES,
      intervalSeconds: TIMESERIES_INTERVAL_SECONDS,
    });
  };
}

function percentileToValue(percentile: string): number {
  const map: Record<string, number> = {
    p50: 0.5,
    p75: 0.75,
    p90: 0.9,
    p95: 0.95,
    p99: 0.99,
  };
  return map[percentile] ?? 0.5;
}

export const deploymentLatencyRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
  percentile: z.string().default("p50"),
});

export const deploymentLatencyResponseSchema = z.object({
  latency: z.number(),
});

export function getDeploymentLatency(ch: Querier) {
  return async (args: z.infer<typeof deploymentLatencyRequestSchema>) => {
    const percentileValue = percentileToValue(args.percentile);

    const query = ch.query({
      query: `
        SELECT
          round(quantile({percentileValue: Float64})(total_latency), 2) as latency
        FROM default.sentinel_requests_raw_v1
        WHERE workspace_id = {workspaceId: String}
          AND project_id = {projectId: String}
          AND deployment_id = {deploymentId: String}
          AND environment_id = {environmentId: String}
          AND time >= toUnixTimestamp(now() - INTERVAL {windowMinutes: UInt16} MINUTE) * 1000`,
      params: deploymentLatencyRequestSchema.extend({
        windowMinutes: z.number(),
        percentileValue: z.number(),
      }),
      schema: deploymentLatencyResponseSchema,
    });

    return query({
      ...args,
      windowMinutes: TIMESERIES_WINDOW_HOURS * 60,
      percentileValue,
    });
  };
}

export const deploymentLatencyTimeseriesRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
  percentile: z.string().default("p50"),
});

export const deploymentLatencyTimeseriesResponseSchema = z.object({
  x: z.number().int(),
  y: z.number(),
});

export function getDeploymentLatencyTimeseries(ch: Querier) {
  return async (args: z.infer<typeof deploymentLatencyTimeseriesRequestSchema>) => {
    const percentileValue = percentileToValue(args.percentile);

    const query = ch.query({
      query: `
        SELECT
          toUnixTimestamp(bucket) * 1000 as x,
          round(quantile({percentileValue: Float64})(total_latency), 2) as y
        FROM (
          SELECT
            toStartOfInterval(toDateTime(time / 1000), INTERVAL {intervalMinutes: UInt8} MINUTE) as bucket,
            total_latency
          FROM default.sentinel_requests_raw_v1
          WHERE workspace_id = {workspaceId: String}
            AND project_id = {projectId: String}
            AND deployment_id = {deploymentId: String}
            AND environment_id = {environmentId: String}
            AND time >= toUnixTimestamp(now() - INTERVAL {windowHours: UInt8} HOUR) * 1000
        )
        GROUP BY bucket
        ORDER BY bucket ASC
        WITH FILL
          FROM toStartOfInterval(now() - INTERVAL {windowHours: UInt8} HOUR, INTERVAL {intervalMinutes: UInt8} MINUTE)
          TO toStartOfInterval(now(), INTERVAL {intervalMinutes: UInt8} MINUTE)
          STEP INTERVAL {intervalMinutes: UInt8} MINUTE`,
      params: deploymentLatencyTimeseriesRequestSchema.extend({
        windowHours: z.number(),
        intervalMinutes: z.number(),
        percentileValue: z.number(),
      }),
      schema: deploymentLatencyTimeseriesResponseSchema,
    });

    return query({
      ...args,
      windowHours: TIMESERIES_WINDOW_HOURS,
      intervalMinutes: TIMESERIES_INTERVAL_MINUTES,
      percentileValue,
    });
  };
}
