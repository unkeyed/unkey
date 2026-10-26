import { z } from "zod";
import type { Querier } from "../client";
import { DAY_MS, type Grid, type Rollup, pickRollup } from "./grid";

export const TRAFFIC_ROLLUPS: readonly Rollup[] = [
  { table: "frontline_requests_per_minute_v1", bucketMs: 60 * 1000, ttlMs: 14 * DAY_MS },
  { table: "frontline_requests_per_5m_v1", bucketMs: 5 * 60 * 1000, ttlMs: 30 * DAY_MS },
  { table: "frontline_requests_per_15m_v1", bucketMs: 15 * 60 * 1000, ttlMs: 30 * DAY_MS },
  { table: "frontline_requests_per_hour_v1", bucketMs: 60 * 60 * 1000, ttlMs: 90 * DAY_MS },
];

export const trafficPoint = z.object({
  t: z.number().int(),
  requests: z.number(),
  errors: z.number().int(),
  errorRate: z.number().nullable(),
  p50: z.number().nullable(),
  p95: z.number().nullable(),
  p99: z.number().nullable(),
});

/**
 * One bucket of an environment's traffic. requests is per second; errors is
 * the 5xx count; errorRate is errors / requests and, like the latency
 * quantiles (ms), is null for a bucket with no requests.
 */
export type TrafficPoint = z.infer<typeof trafficPoint>;

export type AppTrafficRequest = {
  workspaceId: string;
  projectId: string;
  appId: string;
  environmentId: string;
  grid: Grid;
};

const params = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  appId: z.string(),
  environmentId: z.string(),
  table: z.string(),
  bucketSec: z.number().int(),
  startMs: z.number().int(),
  endMs: z.number().int(),
  bucketMs: z.number().int(),
});

const QUERY = `
  SELECT
    t,
    toFloat64(total / {bucketSec: UInt32}) AS requests,
    toUInt64(errors) AS errors,
    if(total = 0, NULL, errors / total) AS errorRate,
    if(total = 0, NULL, p50) AS p50,
    if(total = 0, NULL, p95) AS p95,
    if(total = 0, NULL, p99) AS p99
  FROM (
    SELECT
      toInt64(intDiv(toUnixTimestamp(time), {bucketSec: UInt32}) * {bucketSec: UInt32}) * 1000 AS t,
      sum(count) AS total,
      sumIf(count, response_status >= 500) AS errors,
      quantileTDigestMerge(0.5)(latency_p50) AS p50,
      quantileTDigestMerge(0.95)(latency_p95) AS p95,
      quantileTDigestMerge(0.99)(latency_p99) AS p99
    FROM {table: Identifier}
    WHERE workspace_id = {workspaceId: String}
      AND project_id = {projectId: String}
      AND app_id = {appId: String}
      AND environment_id = {environmentId: String}
      AND time >= fromUnixTimestamp64Milli({startMs: Int64})
      AND time <  fromUnixTimestamp64Milli({endMs: Int64})
    GROUP BY t
  )
  ORDER BY t ASC
  WITH FILL
    FROM {startMs: Int64}
    TO {endMs: Int64}
    STEP {bucketMs: Int64}`;

export function getAppTraffic(ch: Querier) {
  const query = ch.query({ query: QUERY, params, schema: trafficPoint });
  return (req: AppTrafficRequest, nowMs = Date.now()) => {
    const { grid } = req;
    return query({
      workspaceId: req.workspaceId,
      projectId: req.projectId,
      appId: req.appId,
      environmentId: req.environmentId,
      table: pickRollup(TRAFFIC_ROLLUPS, grid, nowMs).table,
      bucketSec: grid.bucketMs / 1000,
      startMs: grid.startMs,
      endMs: grid.endMs,
      bucketMs: grid.bucketMs,
    });
  };
}
