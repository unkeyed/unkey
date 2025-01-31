import { z } from "zod";
import type { Querier } from "./client/interface";
import { dateTimeToUnix } from "./util";

export const getLogsClickhousePayload = z.object({
  workspaceId: z.string(),
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  paths: z
    .array(
      z.object({
        operator: z.enum(["is", "startsWith", "endsWith", "contains"]),
        value: z.string(),
      }),
    )
    .nullable(),
  hosts: z.array(z.string()).nullable(),
  methods: z.array(z.string()).nullable(),
  requestIds: z.array(z.string()).nullable(),
  statusCodes: z.array(z.number().int()).nullable(),
  cursorTime: z.number().int().nullable(),
  cursorRequestId: z.string().nullable(),
});

export const log = z.object({
  request_id: z.string(),
  time: z.number().int(),
  workspace_id: z.string(),
  host: z.string(),
  method: z.string(),
  path: z.string(),
  request_headers: z.array(z.string()),
  request_body: z.string(),
  response_status: z.number().int(),
  response_headers: z.array(z.string()),
  response_body: z.string(),
  error: z.string(),
  service_latency: z.number().int(),
});

export type Log = z.infer<typeof log>;
export type GetLogsClickhousePayload = z.infer<typeof getLogsClickhousePayload>;

export function getLogs(ch: Querier) {
  return async (args: GetLogsClickhousePayload) => {
    // Generate dynamic path conditions
    const pathConditions =
      args.paths
        ?.map((p) => {
          switch (p.operator) {
            case "is":
              return `path = '${p.value}'`;
            case "startsWith":
              return `startsWith(path, '${p.value}')`;
            case "endsWith":
              return `endsWith(path, '${p.value}')`;
            case "contains":
              return `like(path, '%${p.value}%')`;
            default:
              return null;
          }
        })
        .filter(Boolean)
        .join(" OR ") || "TRUE";

    const query = ch.query({
      query: `
        WITH filtered_requests AS (
          SELECT *
          FROM metrics.raw_api_requests_v1
          WHERE workspace_id = {workspaceId: String}
            AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
            
            ---------- Apply request ID filter if present (highest priority)
            AND (
              CASE
                WHEN length({requestIds: Array(String)}) > 0 THEN 
                  request_id IN {requestIds: Array(String)}
                ELSE TRUE
              END
            )
            
            ---------- Apply host filter
            AND (
              CASE
                WHEN length({hosts: Array(String)}) > 0 THEN 
                  host IN {hosts: Array(String)}
                ELSE TRUE
              END
            )
            
            ---------- Apply method filter
            AND (
              CASE
                WHEN length({methods: Array(String)}) > 0 THEN 
                  method IN {methods: Array(String)}
                ELSE TRUE
              END
            )
            
            ---------- Apply path filter using pre-generated conditions
            AND (${pathConditions})
            
            ---------- Apply status code filter
            AND (
              CASE
                WHEN length({statusCodes: Array(UInt16)}) > 0 THEN
                  response_status IN (
                    SELECT status
                    FROM (
                      SELECT multiIf(
                        code = 200, arrayJoin(range(200, 300)),
                        code = 400, arrayJoin(range(400, 500)),
                        code = 500, arrayJoin(range(500, 600)),
                        code
                      ) as status
                      FROM (
                        SELECT arrayJoin({statusCodes: Array(UInt16)}) as code
                      )
                    )
                  )
                ELSE TRUE
              END
            )
            
            -- Apply cursor pagination last
            AND (
              CASE
                WHEN {cursorTime: Nullable(UInt64)} IS NOT NULL 
                  AND {cursorRequestId: Nullable(String)} IS NOT NULL
                THEN (time, request_id) < (
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
          host,
          method,
          path,
          request_headers,
          request_body,
          response_status,
          response_headers,
          response_body,
          error,
          service_latency
        FROM filtered_requests
        ORDER BY time DESC, request_id DESC
        LIMIT {limit: Int}`,
      params: getLogsClickhousePayload,
      schema: log,
    });

    return query(args);
  };
}

export const logsTimeseriesParams = z.object({
  workspaceId: z.string(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  paths: z
    .array(
      z.object({
        operator: z.enum(["is", "startsWith", "endsWith", "contains"]),
        value: z.string(),
      }),
    )
    .nullable(),
  hosts: z.array(z.string()).nullable(),
  methods: z.array(z.string()).nullable(),
  statusCodes: z.array(z.number().int()).nullable(),
});

export const logsTimeseriesDataPoint = z.object({
  x: dateTimeToUnix,
  y: z.object({
    success: z.number().int().default(0),
    error: z.number().int().default(0),
    warning: z.number().int().default(0),
    total: z.number().int().default(0),
  }),
});

export type LogsTimeseriesDataPoint = z.infer<typeof logsTimeseriesDataPoint>;
export type LogsTimeseriesParams = z.infer<typeof logsTimeseriesParams>;

type TimeInterval = {
  table: string;
  timeFunction: string;
  step: string;
};

const INTERVALS: Record<string, TimeInterval> = {
  minute: {
    table: "metrics.api_requests_per_minute_v1",
    timeFunction: "toStartOfMinute",
    step: "MINUTE",
  },
  hour: {
    table: "metrics.api_requests_per_hour_v1",
    timeFunction: "toStartOfHour",
    step: "HOUR",
  },
  day: {
    table: "metrics.api_requests_per_day_v1",
    timeFunction: "toStartOfDay",
    step: "DAY",
  },
} as const;

function createTimeseriesQuery(interval: TimeInterval, whereClause: string) {
  return `
    SELECT
    time as x,
    map(
        'success', SUM(IF(response_status >= 200 AND response_status < 300, count, 0)),
        'warning', SUM(IF(response_status >= 400 AND response_status < 500, count, 0)),
        'error', SUM(IF(response_status >= 500, count, 0)),
        'total', SUM(count)
    ) as y
FROM ${interval.table}
${whereClause}
GROUP BY time
ORDER BY time ASC
WITH FILL
    FROM ${interval.timeFunction}(fromUnixTimestamp64Milli({startTime: Int64}))
    TO ${interval.timeFunction}(fromUnixTimestamp64Milli({endTime: Int64}))
    STEP INTERVAL 1 ${interval.step}  `;
}

function getLogsTimeseriesWhereClause(
  params: LogsTimeseriesParams,
  additionalConditions: string[] = [],
): string {
  const conditions = [
    "workspace_id = {workspaceId: String}",
    // Host filter
    `(CASE
        WHEN length({hosts: Array(String)}) > 0 THEN 
          host IN {hosts: Array(String)}
        ELSE TRUE
      END)`,
    // Method filter
    `(CASE
        WHEN length({methods: Array(String)}) > 0 THEN 
          method IN {methods: Array(String)}
        ELSE TRUE
      END)`,
    // Status code filter
    `(CASE 
        WHEN length({statusCodes: Array(UInt16)}) > 0 THEN
          response_status IN (
            SELECT status 
            FROM (
              SELECT multiIf(
                code = 200, arrayJoin(range(200, 300)),
                code = 400, arrayJoin(range(400, 500)),
                code = 500, arrayJoin(range(500, 600)),
                code
              ) as status 
              FROM (
                SELECT arrayJoin({statusCodes: Array(UInt16)}) as code
              )
            )
          ) 
        ELSE TRUE 
      END)`,
    ...additionalConditions,
  ];

  // Path filter with operators
  if (params.paths?.length) {
    const pathConditions = params.paths
      .map((p) => {
        switch (p.operator) {
          case "is":
            return `path = '${p.value}'`;
          case "startsWith":
            return `startsWith(path, '${p.value}')`;
          case "endsWith":
            return `endsWith(path, '${p.value}')`;
          case "contains":
            return `like(path, '%${p.value}%')`;
        }
      })
      .join(" OR ");
    conditions.push(`(${pathConditions})`);
  }

  return conditions.length > 0 ? `WHERE ${conditions.join(" AND ")}` : "";
}

function createTimeseriesQuerier(interval: TimeInterval) {
  return (ch: Querier) => async (args: LogsTimeseriesParams) => {
    const whereClause = getLogsTimeseriesWhereClause(args, [
      "time >= fromUnixTimestamp64Milli({startTime: Int64})",
      "time <= fromUnixTimestamp64Milli({endTime: Int64})",
    ]);
    const query = createTimeseriesQuery(interval, whereClause);

    return ch.query({
      query,
      params: logsTimeseriesParams,
      schema: logsTimeseriesDataPoint,
    })(args);
  };
}

export const getMinutelyLogsTimeseries = createTimeseriesQuerier(INTERVALS.minute);
export const getHourlyLogsTimeseries = createTimeseriesQuerier(INTERVALS.hour);
export const getDailyLogsTimeseries = createTimeseriesQuerier(INTERVALS.day);
