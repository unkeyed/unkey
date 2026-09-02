import { z } from "zod";
import type { Querier } from "./client";

/** Validates the time range and bucket size for a log drain metrics query. */
export const logdrainMetricsParams = z.object({
  workspaceId: z.string(),
  drainId: z.string(),
  startMs: z.int(),
  endMs: z.int(),
  bucketMinutes: z.int().positive(),
});

/** Validates one filled time bucket from a log drain metrics query. */
export const logdrainMetric = z.object({
  ts: z.int(),
  successCount: z.int(),
  transientErrorCount: z.int(),
  permanentErrorCount: z.int(),
  eventsDelivered: z.int(),
  avgDurationMs: z.number(),
});

/** Parameters for a log drain metrics query. */
export type LogdrainMetricsParams = z.infer<typeof logdrainMetricsParams>;

/** Returns chart metrics with explicit zero values for empty time buckets. */
export function getLogdrainMetrics(ch: Querier) {
  return ch.query({
    query: `
      SELECT
        intDiv(time, {bucketMinutes: UInt16} * 60000) * {bucketMinutes: UInt16} * 60000 AS ts,
        countIf(outcome = 'success') AS successCount,
        countIf(outcome IN ('error', 'transient_error')) AS transientErrorCount,
        countIf(outcome = 'permanent_error') AS permanentErrorCount,
        sumIf(events, outcome = 'success') AS eventsDelivered,
        avg(webhook_duration_ms) AS avgDurationMs
      FROM default.logdrain_deliveries_raw_v1
      PREWHERE workspace_id = {workspaceId: String}
        AND drain_id = {drainId: String}
        AND time >= {startMs: Int64}
        AND time <= {endMs: Int64}
      GROUP BY ts
      ORDER BY ts ASC
      WITH FILL
        FROM intDiv({startMs: Int64}, {bucketMinutes: UInt16} * 60000) * {bucketMinutes: UInt16} * 60000
        TO intDiv({endMs: Int64}, {bucketMinutes: UInt16} * 60000) * {bucketMinutes: UInt16} * 60000 + {bucketMinutes: UInt16} * 60000
        STEP {bucketMinutes: UInt16} * 60000
    `,
    params: logdrainMetricsParams,
    schema: logdrainMetric,
  });
}

/** Validates the scope and result limit for a recent delivery error query. */
export const recentLogdrainErrorsParams = z.object({
  workspaceId: z.string(),
  drainId: z.string(),
  startMs: z.int(),
  limit: z.int().positive().max(100),
});

/** Validates one failed delivery returned by ClickHouse. */
export const recentLogdrainError = z.object({
  time: z.int(),
  outcome: z.enum(["error", "transient_error", "permanent_error"]),
  responseStatus: z.int(),
  responseBody: z.string(),
  error: z.string(),
});

/** Returns the newest failed deliveries for one log drain. */
export function getRecentLogdrainErrors(ch: Querier) {
  return ch.query({
    query: `
      SELECT
        time,
        outcome,
        response_status AS responseStatus,
        response_body AS responseBody,
        error
      FROM default.logdrain_deliveries_raw_v1
      PREWHERE workspace_id = {workspaceId: String}
        AND drain_id = {drainId: String}
        AND time >= {startMs: Int64}
      WHERE outcome != 'success'
      ORDER BY time DESC
      LIMIT {limit: UInt8}
    `,
    params: recentLogdrainErrorsParams,
    schema: recentLogdrainError,
  });
}
