import { z } from "zod";
import type { Querier } from "./client/interface";
import { dateTimeToUnix } from "./util";

export const getLogsClickhousePayload = z.object({
  workspaceId: z.string(),
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  path: z.string().optional().nullable(),
  host: z.string().optional().nullable(),
  requestId: z.string().optional().nullable(),
  method: z.string().optional().nullable(),
  responseStatus: z.array(z.number().int()).nullable(),
});
export type GetLogsClickhousePayload = z.infer<typeof getLogsClickhousePayload>;

export function getLogs(ch: Querier) {
  return async (args: GetLogsClickhousePayload) => {
    const query = ch.query({
      query: `
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
    FROM metrics.raw_api_requests_v1
        WHERE workspace_id = {workspaceId: String}
        AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
        AND (CASE
                WHEN {host: String} != '' THEN host = {host: String}
                ELSE TRUE
            END)
        AND (CASE
                WHEN {requestId: String} != '' THEN request_id = {requestId: String}
                ELSE TRUE
            END)
        AND (CASE
                WHEN {method: String} != '' THEN method = {method: String}
                ELSE TRUE
            END)
        AND (CASE
                WHEN {path: String} != '' THEN path = {path: String}
                ELSE TRUE
            END)
        AND (CASE
                WHEN {responseStatus: Array(UInt16)} IS NOT NULL AND length({responseStatus: Array(UInt16)}) > 0 THEN
                    response_status IN (
                        SELECT status
                        FROM (
                            SELECT
                                multiIf(
                                    code = 200, arrayJoin(range(200, 300)),
                                    code = 400, arrayJoin(range(400, 500)),
                                    code = 500, arrayJoin(range(500, 600)),
                                    code
                                ) as status
                            FROM (
                                SELECT arrayJoin({responseStatus: Array(UInt16)}) as code
                            )
                        )
                    )
                ELSE TRUE
            END)
    ORDER BY time DESC
    LIMIT {limit: Int}`,
      params: getLogsClickhousePayload,
      schema: z.object({
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
      }),
    });
    return query(args);
  };
}

export const logsTimeseriesParams = z.object({
  workspaceId: z.string(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  path: z.string().optional().nullable(),
  host: z.string().optional().nullable(),
  method: z.string().optional().nullable(),
  responseStatus: z.array(z.number().int()).nullable(),
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
    `(CASE 
        WHEN {responseStatus: Array(UInt16)} IS NOT NULL AND length({responseStatus: Array(UInt16)}) > 0 
        THEN response_status IN (
          SELECT status 
          FROM (
            SELECT multiIf(
              code = 200, arrayJoin(range(200, 300)),
              code = 400, arrayJoin(range(400, 500)),
              code = 500, arrayJoin(range(500, 600)),
              code
            ) as status 
            FROM (
              SELECT arrayJoin({responseStatus: Array(UInt16)}) as code
            )
          )
        ) 
        ELSE TRUE 
      END)`,
    ...additionalConditions,
  ];

  if (params.path) {
    conditions.push("path = {path: String}");
  }
  if (params.host) {
    conditions.push("host = {host: String}");
  }
  if (params.method) {
    conditions.push("method = {method: String}");
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
