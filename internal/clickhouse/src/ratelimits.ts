import { z } from "zod";
import type { Inserter, Querier } from "./client";
import { dateTimeToUnix } from "./util";

const params = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  identifier: z.array(z.string()).optional(),
  start: z.number().default(0),
  end: z.number().default(() => Date.now()),
});

export function insertRatelimit(ch: Inserter) {
  return ch.insert({
    table: "ratelimits.raw_ratelimits_v1",
    schema: z.object({
      request_id: z.string(),
      time: z.number().int(),
      workspace_id: z.string(),
      namespace_id: z.string(),
      identifier: z.string(),
      passed: z.boolean(),
    }),
  });
}

export function getRatelimitsPerMinute(ch: Querier) {
  return async (args: z.input<typeof params>) => {
    const query = ch.query({
      query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_minute_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}) > 0" : ""}
      AND time >= fromUnixTimestamp64Milli({start: Int64})
      AND time <= fromUnixTimestamp64Milli({end: Int64})
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfMinute(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfMinute(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 MINUTE
;`,
      params,
      schema: z.object({
        time: dateTimeToUnix,
        passed: z.number(),
        total: z.number(),
      }),
    });

    return query(args);
  };
}

export function getRatelimitsPerHour(ch: Querier) {
  return async (args: z.infer<typeof params>) => {
    const query = ch.query({
      query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_hour_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}) > 0" : ""}
      AND time >= fromUnixTimestamp64Milli({start: Int64})
      AND time <= fromUnixTimestamp64Milli({end: Int64})
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfHour(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfHour(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 HOUR
;`,
      params,
      schema: z.object({
        time: dateTimeToUnix,
        passed: z.number(),
        total: z.number(),
      }),
    });

    return query(args);
  };
}
export function getRatelimitsPerDay(ch: Querier) {
  return async (args: z.infer<typeof params>) => {
    const query = ch.query({
      query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_day_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}) > 0" : ""}
      AND time >= fromUnixTimestamp64Milli({start: Int64})
      AND time <= fromUnixTimestamp64Milli({end: Int64})
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfDay(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfDay(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 DAY
;`,
      params,
      schema: z.object({
        time: dateTimeToUnix,
        passed: z.number(),
        total: z.number(),
      }),
    });

    return query(args);
  };
}
export function getRatelimitsPerMonth(ch: Querier) {
  return async (args: z.input<typeof params>) => {
    const query = ch.query({
      query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_month_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}) > 0" : ""}
      AND time >= fromUnixTimestamp64Milli({start: Int64})
      AND time <= fromUnixTimestamp64Milli({end: Int64})
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfMonth(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfMonth(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 MONTH
;`,
      params,
      schema: z.object({
        time: dateTimeToUnix,
        passed: z.number(),
        total: z.number(),
      }),
    });

    return query(args);
  };
}

const getRatelimitLastUsedParameters = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  identifier: z.array(z.string()).optional(),
  limit: z.number().int(),
});

export function getRatelimitLastUsed(ch: Querier) {
  return async (args: z.input<typeof getRatelimitLastUsedParameters>) => {
    const query = ch.query({
      query: `
    SELECT
      identifier,
      max(time) as time
    FROM ratelimits.ratelimits_last_used_v1
    WHERE
      workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
     ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}) > 0" : ""}
    GROUP BY identifier
    ORDER BY time DESC
    LIMIT {limit: Int}
;`,
      params: getRatelimitLastUsedParameters,
      schema: z.object({
        identifier: z.string(),
        time: z.number(),
      }),
    });

    return query(args);
  };
}

// ------------------------------------------  LOGS-V2
export const ratelimitLogsParams = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  requestIds: z.array(z.string()).nullable(),
  identifiers: z
    .array(
      z.object({
        operator: z.enum(["is", "contains"]),
        value: z.string(),
      }),
    )
    .nullable(),
  rejected: z.number().int().nullable(),
  cursorTime: z.number().int().nullable(),
  cursorRequestId: z.string().nullable(),
});

export const ratelimitLogs = z.object({
  request_id: z.string(),
  time: z.number().int(),
  identifier: z.string(),
  rejected: z.number().int(),

  // Fields from metrics table
  host: z.string(),
  method: z.string(),
  path: z.string(),
  request_headers: z.array(z.string()),
  request_body: z.string(),
  response_status: z.number().int(),
  response_headers: z.array(z.string()),
  response_body: z.string(),
  service_latency: z.number().int(),
  user_agent: z.string(),
  colo: z.string(),
});

export type RatelimitLog = z.infer<typeof ratelimitLogs>;
export type RatelimitLogsParams = z.infer<typeof ratelimitLogsParams>;

export function getRatelimitLogs(ch: Querier) {
  return async (args: RatelimitLogsParams) => {
    const identifierConditions =
      args.identifiers
        ?.map((p) => {
          switch (p.operator) {
            case "is":
              return `identifier = '${p.value}'`;
            case "contains":
              return `position(identifier, '${p.value}') > 0`;
            default:
              return null;
          }
        })
        .filter(Boolean)
        .join(" OR ") || "TRUE";

    const query = ch.query({
      query: `
        WITH filtered_requests AS (
          SELECT 
            -- Rate limits fields
            r.request_id,
            r.time,
            r.workspace_id,   
            r.namespace_id,  
            r.identifier,
            r.passed,
            
            -- Metrics fields
            m.host,
            m.method,
            m.path,
            m.request_headers,
            m.request_body,
            m.response_status,
            m.response_headers,
            m.response_body,
            m.service_latency,
            m.user_agent,
            m.colo
          FROM ratelimits.raw_ratelimits_v1 r
          LEFT JOIN metrics.raw_api_requests_v1 m ON 
            r.request_id = m.request_id
          WHERE r.workspace_id = {workspaceId: String}
            AND r.namespace_id = {namespaceId: String}
            AND r.time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
            ---------- Apply request ID filter if present (highest priority)
            AND (
              CASE
                WHEN length({requestIds: Array(String)}) > 0 THEN 
                  r.request_id IN {requestIds: Array(String)}
                ELSE TRUE
              END
            )
            
            ---------- Apply identifier filter
            AND (${identifierConditions})
            
            ---------- Apply rejected filter
            AND (
              CASE
                WHEN {rejected: Nullable(UInt8)} IS NOT NULL THEN 
                  (NOT r.passed) = {rejected: Nullable(UInt8)}
                ELSE TRUE
              END
            )
            
            -- Apply cursor pagination last
            AND (
              CASE
                WHEN {cursorTime: Nullable(UInt64)} IS NOT NULL 
                  AND {cursorRequestId: Nullable(String)} IS NOT NULL
                THEN (r.time, r.request_id) < (
                  {cursorTime: Nullable(UInt64)}, 
                  {cursorRequestId: Nullable(String)}
                )
                ELSE TRUE
              END
            )
        )
        
        SELECT
          request_id,
          time,
          workspace_id,
          namespace_id,
          identifier,
          toUInt8(NOT passed) as rejected,
          host,
          method,
          path,
          request_headers,
          request_body,
          response_status,
          response_headers,
          response_body,
          service_latency,
          user_agent,
          colo
        FROM filtered_requests
        ORDER BY time DESC, request_id DESC
        LIMIT {limit: Int}`,
      params: ratelimitLogsParams,
      schema: ratelimitLogs,
    });
    return query(args);
  };
}
