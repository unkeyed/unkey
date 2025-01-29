import { z } from "zod";
import type { Inserter, Querier } from "./client";
import { dateTimeToUnix } from "./util";

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

export const ratelimitLogsTimeseriesParams = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  identifiers: z
    .array(
      z.object({
        operator: z.enum(["is", "contains"]),
        value: z.string(),
      }),
    )
    .nullable(),
});

export const ratelimitLogsTimeseriesDataPoint = z.object({
  x: dateTimeToUnix,
  y: z.object({
    passed: z.number().int().default(0),
    total: z.number().int().default(0),
  }),
});

export type RatelimitLogsTimeseriesDataPoint = z.infer<typeof ratelimitLogsTimeseriesDataPoint>;
export type RatelimitLogsTimeseriesParams = z.infer<typeof ratelimitLogsTimeseriesParams>;

type TimeInterval = {
  table: string;
  timeFunction: string;
  step: string;
};

const INTERVALS: Record<string, TimeInterval> = {
  minute: {
    table: "ratelimits.ratelimits_per_minute_v1",
    timeFunction: "toStartOfMinute",
    step: "MINUTE",
  },
  hour: {
    table: "ratelimits.ratelimits_per_hour_v1",
    timeFunction: "toStartOfHour",
    step: "HOUR",
  },
  day: {
    table: "ratelimits.ratelimits_per_day_v1",
    timeFunction: "toStartOfDay",
    step: "DAY",
  },
  month: {
    table: "ratelimits.ratelimits_per_month_v1",
    timeFunction: "toStartOfMonth",
    step: "MONTH",
  },
} as const;

function createTimeseriesQuery(interval: TimeInterval, whereClause: string) {
  return `
    SELECT
      time as x,
      map(
        'passed', sum(passed),
        'total', sum(total)
      ) as y
    FROM ${interval.table}
    ${whereClause}
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM ${interval.timeFunction}(fromUnixTimestamp64Milli({startTime: Int64}))
      TO ${interval.timeFunction}(fromUnixTimestamp64Milli({endTime: Int64}))
      STEP INTERVAL 1 ${interval.step}
  `;
}

function getRatelimitLogsTimeseriesWhereClause(
  params: RatelimitLogsTimeseriesParams,
  additionalConditions: string[] = [],
): string {
  const conditions = [
    "workspace_id = {workspaceId: String}",
    "namespace_id = {namespaceId: String}",
    ...additionalConditions,
  ];

  // Identifier filter with operators
  if (params.identifiers?.length) {
    const identifierConditions = params.identifiers
      .map((i) => {
        switch (i.operator) {
          case "is":
            return `identifier = '${i.value}'`;
          case "contains":
            return `like(identifier, '%${i.value}%')`;
        }
      })
      .join(" OR ");
    conditions.push(`(${identifierConditions})`);
  }

  return conditions.length > 0 ? `WHERE ${conditions.join(" AND ")}` : "";
}

function createTimeseriesQuerier(interval: TimeInterval) {
  return (ch: Querier) => async (args: RatelimitLogsTimeseriesParams) => {
    const whereClause = getRatelimitLogsTimeseriesWhereClause(args, [
      "time >= fromUnixTimestamp64Milli({startTime: Int64})",
      "time <= fromUnixTimestamp64Milli({endTime: Int64})",
    ]);
    const query = createTimeseriesQuery(interval, whereClause);

    return ch.query({
      query,
      params: ratelimitLogsTimeseriesParams,
      schema: ratelimitLogsTimeseriesDataPoint,
    })(args);
  };
}

export const getRatelimitsPerMinute = createTimeseriesQuerier(INTERVALS.minute);
export const getRatelimitsPerHour = createTimeseriesQuerier(INTERVALS.hour);
export const getRatelimitsPerDay = createTimeseriesQuerier(INTERVALS.day);
export const getRatelimitsPerMonth = createTimeseriesQuerier(INTERVALS.month);

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
