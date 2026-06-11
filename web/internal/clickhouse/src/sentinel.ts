import { z } from "zod";
import type { Querier } from "./client";

export const TIMESERIES_WINDOW_HOURS = 6;
export const TIMESERIES_INTERVAL_MINUTES = 15;
const TIMESERIES_INTERVAL_SECONDS = TIMESERIES_INTERVAL_MINUTES * 60;
const CURRENT_RPS_WINDOW_MINUTES = 15;
const CURRENT_RPS_WINDOW_MS = CURRENT_RPS_WINDOW_MINUTES * 60 * 1000;

const TABLE = "default.sentinel_requests_raw_v1";
const MV_TABLE = "default.sentinel_requests_per_15m_v1";

const SQL = {
  deploymentFilter: `
    workspace_id = {workspaceId: String}
    AND project_id = {projectId: String}
    AND deployment_id = {deploymentId: String}
    AND environment_id = {environmentId: String}`,

  // time (ms) >= X minutes ago (ms)
  recentMinutes: "time >= toUnixTimestamp(now() - INTERVAL {windowMinutes: UInt16} MINUTE) * 1000",

  // MV time (DateTime) >= X hours ago  -- for MV (DateTime seconds)
  mvRecentHours: "time >= now() - INTERVAL {windowHours: UInt8} HOUR",

  // Fill gaps in timeseries with zeros
  fillGaps: `
    WITH FILL
      FROM toStartOfInterval(now() - INTERVAL {windowHours: UInt8} HOUR, INTERVAL {intervalMinutes: UInt8} MINUTE)
      TO toStartOfInterval(now(), INTERVAL {intervalMinutes: UInt8} MINUTE)
      STEP INTERVAL {intervalMinutes: UInt8} MINUTE`,
} as const;

type PercentileKey = "p50" | "p75" | "p90" | "p95" | "p99";

const LATENCY_MERGE: Record<PercentileKey, string> = {
  p50: "round(quantileTDigestMerge(0.5)(latency_p50), 2)",
  p75: "round(quantileTDigestMerge(0.75)(latency_p75), 2)",
  p90: "round(quantileTDigestMerge(0.9)(latency_p90), 2)",
  p95: "round(quantileTDigestMerge(0.95)(latency_p95), 2)",
  p99: "round(quantileTDigestMerge(0.99)(latency_p99), 2)",
};

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
// quantile() returns NULL when the time window contains zero rows (no
// traffic in the last N minutes). Allow null at the schema layer and let
// callers coalesce to 0; the previous z.number() crashed the request.
const timeseriesPointSchema = z.object({ x: z.number().int(), y: z.number() });

// ─────────────────────────────────────────────────────────────
// Logs
// ─────────────────────────────────────────────────────────────

export const sentinelLogsRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string().nullable().default(null),
  environmentId: z.array(z.string()).default([]),
  limit: z.number().int().positive().default(50),
  startTime: z.int().nullable().default(null),
  endTime: z.int().nullable().default(null),
  since: z.string().nullable().default(null),
  statusCodes: z.array(z.number().int()).nullable().default(null),
  methods: z.array(z.string()).nullable().default(null),
  paths: z
    .array(
      z.object({
        operator: z.literal("contains"),
        value: z.string(),
      }),
    )
    .nullable()
    .default(null),
  cursor: z.number().int().nullable().optional(),
  // 1-based page for offset pagination. Defaults to 1 (offset 0) so cursor-only
  // callers (e.g. the deployment logs view) keep their existing behavior.
  page: z.number().int().min(1).default(1),
});

export type SentinelLogsRequest = z.infer<typeof sentinelLogsRequestSchema>;

export const sentinelLogsResponseSchema = z.object({
  request_id: z.string(),
  time: z.number().int(),
  deployment_id: z.string(),
  region: z.string(),
  method: z.string(),
  path: z.string(),
  host: z.string(),
  response_status: z.number().int(),
  total_latency: z.number().int(),
  instance_latency: z.number().int(),
  sentinel_latency: z.number().int(),
  query_string: z.string(),
  query_params: z.record(z.string(), z.array(z.string())),
  request_headers: z.array(z.string()),
  request_body: z.string(),
  response_headers: z.array(z.string()),
  response_body: z.string(),
  user_agent: z.string(),
  ip_address: z.string(),
});

export type SentinelLogsResponse = z.infer<typeof sentinelLogsResponseSchema>;

export function getSentinelLogs(ch: Querier) {
  return async (args: SentinelLogsRequest) => {
    // Build path filter conditions
    let pathConditions = "TRUE";
    const pathParams: Record<string, z.ZodString> = {};

    if (args.paths && args.paths.length > 0) {
      const conditions = args.paths.map((_, i) => {
        const key = `pathValue${i}`;
        pathParams[key] = z.string();
        return `position(path, {${key}: String}) > 0`;
      });
      pathConditions = `(${conditions.join(" OR ")})`;
    }

    // Build base filters (workspace + project only)
    const baseFilter = `
      workspace_id = {workspaceId: String}
      AND project_id = {projectId: String}
    `;

    const filterConditions = `
      ${baseFilter}
      AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
      AND (CASE WHEN {deploymentId: Nullable(String)} IS NOT NULL
           THEN position(deployment_id, {deploymentId: Nullable(String)}) > 0
           ELSE TRUE END)
      AND (CASE WHEN length({environmentId: Array(String)}) > 0
           THEN environment_id IN {environmentId: Array(String)}
           ELSE TRUE END)
      -- Class codes (200,300,400,500) match via intDiv: intDiv(404,100)*100=400.
      -- Specific codes (401,503) match via exact IN; their class won't be in the array.
      AND (CASE WHEN length({statusCodes: Array(Int32)}) > 0
           THEN (
             response_status IN {statusCodes: Array(Int32)}
             OR intDiv(response_status, 100) * 100 IN {statusCodes: Array(Int32)}
           )
           ELSE TRUE END)
      AND (CASE WHEN length({methods: Array(String)}) > 0
           THEN method IN {methods: Array(String)}
           ELSE TRUE END)
      AND ${pathConditions}
    `;

    // Convert path params to actual values
    const pathValues: Record<string, string> = {};
    if (args.paths) {
      args.paths.forEach((p, i) => {
        pathValues[`pathValue${i}`] = p.value;
      });
    }

    const totalQuery = ch.query({
      query: `SELECT count(*) as total_count FROM ${TABLE} WHERE ${filterConditions}`,
      params: sentinelLogsRequestSchema.extend(
        Object.fromEntries(Object.keys(pathParams).map((k) => [k, z.string()])),
      ),
      schema: z.object({ total_count: z.number().int() }),
    });

    // Offset pagination. `page` is 1-based; cursor-only callers leave it at 1
    // (offset 0). The cursor clause still composes for time-window callers.
    const offset = (args.page - 1) * args.limit;

    const logsQuery = ch.query({
      query: `
        SELECT request_id, time, deployment_id, region, method, path, host,
               response_status, total_latency, instance_latency, sentinel_latency,
               query_string, query_params, request_headers, request_body,
               response_headers, response_body, user_agent, ip_address
        FROM ${TABLE}
        WHERE ${filterConditions}
          AND ({cursor: Nullable(UInt64)} IS NULL OR time < {cursor: Nullable(UInt64)})
        ORDER BY time DESC, request_id DESC
        LIMIT {limit: Int}
        OFFSET {offset: Int}`,
      params: sentinelLogsRequestSchema.extend({
        offset: z.number().int(),
        ...Object.fromEntries(Object.keys(pathParams).map((k) => [k, z.string()])),
      }),
      schema: sentinelLogsResponseSchema,
    });

    return {
      totalQuery: totalQuery({ ...args, ...pathValues } as never),
      logsQuery: logsQuery({ ...args, ...pathValues, offset } as never),
    };
  };
}

// ─────────────────────────────────────────────────────────────
// Region / Instance RPS
// ─────────────────────────────────────────────────────────────

export const regionRpsRequestSchema = baseDeploymentParams.extend({
  region: z.string(),
});

// Avg RPS for a region within a deployment over the rolling current-RPS
// window. The table stores requests per instance, so summing over a region
// is just COUNT() filtered by region — no sentinel join needed.
export function getRegionRps(ch: Querier) {
  return async (args: z.infer<typeof regionRpsRequestSchema>) => {
    const query = ch.query({
      query: `
        SELECT round(COUNT(*) * 1000.0 / {windowMs: UInt64}, 2) as avg_rps
        FROM ${TABLE}
        WHERE ${SQL.deploymentFilter}
          AND region = {region: String}
          AND ${SQL.recentMinutes}`,
      params: regionRpsRequestSchema.extend({
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
// Deployment RPS Timeseries
// ─────────────────────────────────────────────────────────────

export const deploymentRpsTimeseriesRequestSchema = baseDeploymentParams;
export type DeploymentRpsTimeseriesRequest = z.infer<typeof deploymentRpsTimeseriesRequestSchema>;

export function getDeploymentRpsTimeseries(ch: Querier) {
  return async (args: DeploymentRpsTimeseriesRequest) => {
    const query = ch.query({
      query: `
        SELECT
          toUnixTimestamp(time) * 1000 as x,
          round(sum(count) / {intervalSeconds: UInt32}, 2) as y
        FROM ${MV_TABLE}
        WHERE ${SQL.deploymentFilter}
          AND ${SQL.mvRecentHours}
        GROUP BY time
        ORDER BY time ASC
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
// Deployment Latency (current + timeseries in 1 query via ROLLUP)
// ─────────────────────────────────────────────────────────────

export const deploymentLatencyRequestSchema = baseDeploymentParams.extend({
  percentile: percentileSchema,
});

export type DeploymentLatencyRequest = z.infer<typeof deploymentLatencyRequestSchema>;

export function getDeploymentLatencyWithTimeseries(ch: Querier) {
  return async (args: DeploymentLatencyRequest) => {
    const mergeExpr = LATENCY_MERGE[args.percentile];

    const query = ch.query({
      query: `
        SELECT
          toUnixTimestamp(time) * 1000 as x,
          ${mergeExpr} as y
        FROM ${MV_TABLE}
        WHERE ${SQL.deploymentFilter}
          AND ${SQL.mvRecentHours}
        GROUP BY time WITH ROLLUP
        ORDER BY time ASC`,
      params: baseDeploymentParams.extend({
        windowHours: z.number(),
      }),
      schema: timeseriesPointSchema,
    });

    return query({
      ...args,
      windowHours: TIMESERIES_WINDOW_HOURS,
    });
  };
}
