import { z } from "zod";
import type { Querier } from "./client/interface";

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
    const paramSchemaExtension: Record<string, z.ZodString> = {};
    const parameters: Record<string, any> = { ...args };

    // Generate dynamic path conditions with parameterization
    const pathConditions =
      args.paths
        ?.map((p, index) => {
          const paramName = `pathValue_${index}`;
          paramSchemaExtension[paramName] = z.string();
          parameters[paramName] = p.value;

          switch (p.operator) {
            case "is":
              return `path = {${paramName}: String}`;
            case "startsWith":
              return `startsWith(path, {${paramName}: String})`;
            case "endsWith":
              return `endsWith(path, {${paramName}: String})`;
            case "contains":
              return `like(path, CONCAT('%', {${paramName}: String}, '%'))`;
            default:
              return null;
          }
        })
        .filter(Boolean)
        .join(" OR ") || "TRUE";

    const extendedParamsSchema = getLogsClickhousePayload.extend(paramSchemaExtension);

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
      params: extendedParamsSchema,
      schema: log,
    });

    return query(parameters);
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
  x: z.number().int(),
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
  step: string;
  stepSize: number;
};

const INTERVALS: Record<string, TimeInterval> = {
  minute: {
    table: "metrics.api_requests_per_minute_v1",
    step: "MINUTE",
    stepSize: 1,
  },
  fiveMinutes: {
    table: "metrics.api_requests_per_minute_v1",
    step: "MINUTES",
    stepSize: 5,
  },
  fifteenMinutes: {
    table: "metrics.api_requests_per_minute_v1",
    step: "MINUTES",
    stepSize: 15,
  },
  thirtyMinutes: {
    table: "metrics.api_requests_per_minute_v1",
    step: "MINUTES",
    stepSize: 30,
  },
  hour: {
    table: "metrics.api_requests_per_hour_v1",
    step: "HOUR",
    stepSize: 1,
  },
  twoHours: {
    table: "metrics.api_requests_per_hour_v1",
    step: "HOURS",
    stepSize: 2,
  },
  fourHours: {
    table: "metrics.api_requests_per_hour_v1",
    step: "HOURS",
    stepSize: 4,
  },
  sixHours: {
    table: "metrics.api_requests_per_hour_v1",
    step: "HOURS",
    stepSize: 6,
  },
  day: {
    table: "metrics.api_requests_per_day_v1",
    step: "DAY",
    stepSize: 1,
  },
} as const;

function createTimeseriesQuery(interval: TimeInterval, whereClause: string) {
  // For SQL interval definitions
  const intervalUnit = {
    MINUTE: "minute",
    MINUTES: "minute",
    HOUR: "hour",
    HOURS: "hour",
    DAY: "day",
    MONTH: "month",
  }[interval.step];

  // For millisecond step calculation
  const msPerUnit = {
    MINUTE: 60_000,
    MINUTES: 60_000,
    HOUR: 3600_000,
    HOURS: 3600_000,
    DAY: 86400_000,
    MONTH: 2592000_000,
  }[interval.step];

  const stepMs = msPerUnit! * interval.stepSize;

  return `
    SELECT
      toUnixTimestamp64Milli(CAST(toStartOfInterval(time, INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3))) as x,
      map(
          'success', SUM(IF(response_status >= 200 AND response_status < 300, count, 0)),
          'warning', SUM(IF(response_status >= 400 AND response_status < 500, count, 0)),
          'error', SUM(IF(response_status >= 500, count, 0)),
          'total', SUM(count)
      ) as y
    FROM ${interval.table}
    ${whereClause}
    GROUP BY x
    ORDER BY x ASC
    WITH FILL
      FROM toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({startTime: Int64})), INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3)))
      TO toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({endTime: Int64})), INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3)))
      STEP ${stepMs}`;
}

function getLogsTimeseriesWhereClause(
  params: LogsTimeseriesParams,
  additionalConditions: string[] = [],
): { whereClause: string; paramSchema: z.ZodType<any> } {
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

  const paramSchemaExtension: Record<string, z.ZodString> = {};

  // Path filter with parameterized operators
  if (params.paths?.length) {
    const pathConditions = params.paths
      .map((p, index) => {
        const paramName = `pathValue_${index}`;
        paramSchemaExtension[paramName] = z.string();

        switch (p.operator) {
          case "is":
            return `path = {${paramName}: String}`;
          case "startsWith":
            return `startsWith(path, {${paramName}: String})`;
          case "endsWith":
            return `endsWith(path, {${paramName}: String})`;
          case "contains":
            return `like(path, CONCAT('%', {${paramName}: String}, '%'))`;
        }
      })
      .join(" OR ");
    conditions.push(`(${pathConditions})`);
  }

  return {
    whereClause: conditions.length > 0 ? `WHERE ${conditions.join(" AND ")}` : "",
    paramSchema: logsTimeseriesParams.extend(paramSchemaExtension),
  };
}

function createTimeseriesQuerier(interval: TimeInterval) {
  return (ch: Querier) => async (args: LogsTimeseriesParams) => {
    const { whereClause, paramSchema } = getLogsTimeseriesWhereClause(args, [
      "time >= fromUnixTimestamp64Milli({startTime: Int64})",
      "time <= fromUnixTimestamp64Milli({endTime: Int64})",
    ]);

    const parameters = {
      ...args,
      ...(args.paths?.reduce(
        (acc, p, index) => ({
          // biome-ignore lint/performance/noAccumulatingSpread: it's okay to spread
          ...acc,
          [`pathValue_${index}`]: p.value,
        }),
        {},
      ) ?? {}),
    };

    const query = createTimeseriesQuery(interval, whereClause);

    return ch.query({
      query,
      params: paramSchema,
      schema: logsTimeseriesDataPoint,
    })(parameters);
  };
}

export const getMinutelyLogsTimeseries = createTimeseriesQuerier(INTERVALS.minute);
export const getFiveMinuteLogsTimeseries = createTimeseriesQuerier(INTERVALS.fiveMinutes);
export const getFifteenMinuteLogsTimeseries = createTimeseriesQuerier(INTERVALS.fifteenMinutes);
export const getThirtyMinuteLogsTimeseries = createTimeseriesQuerier(INTERVALS.thirtyMinutes);
export const getHourlyLogsTimeseries = createTimeseriesQuerier(INTERVALS.hour);
export const getTwoHourlyLogsTimeseries = createTimeseriesQuerier(INTERVALS.twoHours);
export const getFourHourlyLogsTimeseries = createTimeseriesQuerier(INTERVALS.fourHours);
export const getSixHourlyLogsTimeseries = createTimeseriesQuerier(INTERVALS.sixHours);
export const getDailyLogsTimeseries = createTimeseriesQuerier(INTERVALS.day);
