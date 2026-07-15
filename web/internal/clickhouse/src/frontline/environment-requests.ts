import { z } from "zod";
import type { Querier } from "../client";

// Environment-scoped Frontline request metrics: per-bucket request and error
// counts for a single app and environment, read from the pre-aggregated
// frontline_requests_per_* tables. Deployment-scoped request helpers live in
// sentinel.ts.

// Interval is the bucket-resolution token callers request. Each maps to a bucket
// size in [INTERVAL_MS] and a pre-aggregated table in [TABLE_BY_INTERVAL].
export type Interval = "1m" | "5m" | "15m" | "1h";

// INTERVAL_MS is the bucket size for each interval. It is exported so callers
// can align their grid (floor bounds, derive rates) to the same bucket size
// without learning the aggregate table names, which stay private here.
export const INTERVAL_MS = {
  "1m": 60 * 1000,
  "5m": 5 * 60 * 1000,
  "15m": 15 * 60 * 1000,
  "1h": 60 * 60 * 1000,
} satisfies Record<Interval, number>;

// TABLE_BY_INTERVAL maps each interval to its fully qualified aggregate table.
const TABLE_BY_INTERVAL = {
  "1m": "frontline_requests_per_minute_v1",
  "5m": "frontline_requests_per_5m_v1",
  "15m": "frontline_requests_per_15m_v1",
  "1h": "frontline_requests_per_hour_v1",
} satisfies Record<Interval, string>;

// environmentRequestsParamsSchema validates one environment-level request
// metric query. The caller supplies the interval and grid; this module resolves
// the aggregate table and bucket size from the interval.
export const environmentRequestsParamsSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  appId: z.string(),
  environmentId: z.string(),
  // interval selects the aggregate table and bucket size. The grid
  // [startTimeMs, endTimeMs) is half-open Unix ms, already floored to that
  // bucket size by the caller.
  interval: z.enum(["1m", "5m", "15m", "1h"]),
  startTimeMs: z.number().int().nonnegative(),
  endTimeMs: z.number().int().nonnegative(),
});

// EnvironmentRequestsParams is the input contract for [getEnvironmentRequests].
export type EnvironmentRequestsParams = z.infer<typeof environmentRequestsParamsSchema>;

// environmentRequestsPointSchema is one row of the returned series: `t` is the
// bucket start as Unix ms, `requests` is the bucket's request count, and `errors`
// counts its responses with status >= 500.
const environmentRequestsPointSchema = z.object({
  t: z.number().int(),
  requests: z.number().int(),
  errors: z.number().int(),
});

// getEnvironmentRequests returns one request and error count per bucket for a
// single app and environment over the half-open window [startTimeMs,
// endTimeMs), resolving the aggregate table and bucket size from
// [EnvironmentRequestsParams].interval.
//
// The series is dense: buckets with no traffic come back as zero rows, so a
// caller can render a continuous chart without filling gaps itself. For interval
// "1h" over a three-hour window with traffic only in the first hour, the result
// is three rows: one with real counts and two zeroed.
export function getEnvironmentRequests(ch: Querier) {
  return async (args: EnvironmentRequestsParams) => {
    const query = ch.query({
      query: `
        SELECT
          toInt64(toUnixTimestamp(time) * 1000) AS t,
          toUInt64(sum(count)) AS requests,
          toUInt64(sumIf(count, response_status >= 500)) AS errors
        FROM {tableName: Identifier}
        WHERE workspace_id = {workspaceId: String}
          AND project_id = {projectId: String}
          AND app_id = {appId: String}
          AND environment_id = {environmentId: String}
          AND time >= fromUnixTimestamp64Milli({startTimeMs: Int64})
          AND time <  fromUnixTimestamp64Milli({endTimeMs: Int64})
        GROUP BY t
        ORDER BY t ASC
        WITH FILL
          FROM {startTimeMs: Int64}
          TO {endTimeMs: Int64}
          STEP {bucketMs: Int64}`,
      params: environmentRequestsParamsSchema.extend({
        tableName: z.string(),
        bucketMs: z.number().int(),
      }),
      schema: environmentRequestsPointSchema,
    });

    return query({
      ...args,
      tableName: TABLE_BY_INTERVAL[args.interval],
      bucketMs: INTERVAL_MS[args.interval],
    });
  };
}
