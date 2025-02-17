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
): { whereClause: string; paramSchema: z.ZodType<any> } {
  const conditions = [
    "workspace_id = {workspaceId: String}",
    "namespace_id = {namespaceId: String}",
    ...additionalConditions,
  ];

  const paramSchemaExtension: Record<string, z.ZodString> = {};

  if (params.identifiers?.length) {
    const identifierConditions = params.identifiers
      .map((i, index) => {
        const paramName = `identifierValue_${index}`;
        paramSchemaExtension[paramName] = z.string();

        switch (i.operator) {
          case "is":
            return `identifier = {${paramName}: String}`;
          case "contains":
            return `like(identifier, CONCAT('%', {${paramName}: String}, '%'))`;
        }
      })
      .join(" OR ");
    conditions.push(`(${identifierConditions})`);
  }

  return {
    whereClause: conditions.length > 0 ? `WHERE ${conditions.join(" AND ")}` : "",
    paramSchema: ratelimitLogsTimeseriesParams.extend(paramSchemaExtension),
  };
}

function createTimeseriesQuerier(interval: TimeInterval) {
  return (ch: Querier) => async (args: RatelimitLogsTimeseriesParams) => {
    const { whereClause, paramSchema } = getRatelimitLogsTimeseriesWhereClause(args, [
      "time >= fromUnixTimestamp64Milli({startTime: Int64})",
      "time <= fromUnixTimestamp64Milli({endTime: Int64})",
    ]);

    const parameters = {
      ...args,
      ...(args.identifiers?.reduce(
        (acc, i, index) => ({
          // biome-ignore lint/performance/noAccumulatingSpread: it's okay here
          ...acc,
          [`identifierValue_${index}`]: i.value,
        }),
        {},
      ) ?? {}),
    };

    return ch.query({
      query: createTimeseriesQuery(interval, whereClause),
      params: paramSchema,
      schema: ratelimitLogsTimeseriesDataPoint,
    })(parameters);
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
  status: z
    .array(
      z.object({
        value: z.enum(["blocked", "passed"]),
        operator: z.literal("is"),
      }),
    )
    .nullable(),
  cursorTime: z.number().int().nullable(),
  cursorRequestId: z.string().nullable(),
});

export const ratelimitLogs = z.object({
  request_id: z.string(),
  time: z.number().int(),
  identifier: z.string(),
  status: z.number().int(),

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

interface ExtendedParams extends RatelimitLogsParams {
  [key: string]: unknown;
}

export function getRatelimitLogs(ch: Querier) {
  return async (args: RatelimitLogsParams) => {
    const paramSchemaExtension: Record<string, z.ZodType> = {};
    const parameters: ExtendedParams = { ...args };

    const hasRequestIds = args.requestIds && args.requestIds.length > 0;
    const hasStatusFilters = args.status && args.status.length > 0;
    const hasIdentifierFilters = args.identifiers && args.identifiers.length > 0;

    const statusCondition = !hasStatusFilters
      ? "TRUE"
      : args.status
          ?.map((filter, index) => {
            if (filter.operator === "is") {
              const paramName = `statusValue_${index}`;
              paramSchemaExtension[paramName] = z.boolean();
              parameters[paramName] = filter.value === "passed";
              return `passed = {${paramName}: Boolean}`;
            }
            return null;
          })
          .filter(Boolean)
          .join(" OR ") || "TRUE";

    const identifierConditions = !hasIdentifierFilters
      ? "TRUE"
      : args.identifiers
          ?.map((p, index) => {
            const paramName = `identifierValue_${index}`;
            paramSchemaExtension[paramName] = z.string();
            parameters[paramName] = p.value;
            switch (p.operator) {
              case "is":
                return `identifier = {${paramName}: String}`;
              case "contains":
                return `position({${paramName}: String}, identifier) > 0`;
              default:
                return null;
            }
          })
          .filter(Boolean)
          .join(" OR ") || "TRUE";

    const extendedParamsSchema = ratelimitLogsParams.extend(paramSchemaExtension);

    const query = ch.query({
      query: `
WITH filtered_ratelimits AS (
    SELECT
        request_id,
        time,
        workspace_id,
        namespace_id,
        identifier,
        toUInt8(passed) as status
    FROM ratelimits.raw_ratelimits_v1 r
    WHERE workspace_id = {workspaceId: String}
        AND namespace_id = {namespaceId: String}
        AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
        ${hasRequestIds ? "AND request_id IN {requestIds: Array(String)}" : ""}
        AND (${identifierConditions})
        AND (${statusCondition})
        AND (({cursorTime: Nullable(UInt64)} IS NULL AND {cursorRequestId: Nullable(String)} IS NULL) 
             OR (time, request_id) < ({cursorTime: Nullable(UInt64)}, {cursorRequestId: Nullable(String)}))
)
SELECT 
    fr.request_id,
    fr.time,
    fr.workspace_id,
    fr.namespace_id,
    fr.identifier,
    fr.status,
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
FROM filtered_ratelimits fr
LEFT JOIN (
    SELECT 
        request_id,
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
    FROM metrics.raw_api_requests_v1
    WHERE workspace_id = {workspaceId: String}
        AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
) m ON fr.request_id = m.request_id
ORDER BY fr.time DESC, fr.request_id DESC
LIMIT {limit: Int}`,
      params: extendedParamsSchema,
      schema: ratelimitLogs,
    });

    return query(parameters);
  };
}

// ## OVERVIEWS
export const ratelimitOverviewLogsParams = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  status: z
    .array(
      z.object({
        value: z.enum(["blocked", "passed"]),
        operator: z.literal("is"),
      }),
    )
    .nullable(),
  identifiers: z
    .array(
      z.object({
        operator: z.enum(["is", "contains"]),
        value: z.string(),
      }),
    )
    .nullable(),
  cursorTime: z.number().int().nullable(),
  cursorRequestId: z.string().nullable(),
});

export const ratelimitOverviewLogs = z.object({
  time: z.number().int(),
  identifier: z.string(),
  request_id: z.string(),
  passed_count: z.number().int(),
  blocked_count: z.number().int(),
  avg_latency: z.number().int(),
  p99_latency: z.number().int(),
  override: z
    .object({
      limit: z.number().int(),
      duration: z.number().int(),
      overrideId: z.string(),
      async: z.boolean().nullable(),
    })
    .optional(),
});

export type RatelimitOverviewLog = z.infer<typeof ratelimitOverviewLogs>;
export type RatelimitOverviewLogsParams = z.infer<typeof ratelimitOverviewLogsParams>;

interface ExtendedParamsOverviewLogs extends RatelimitOverviewLogsParams {
  [key: string]: unknown;
}

export function getRatelimitOverviewLogs(ch: Querier) {
  return async (args: RatelimitOverviewLogsParams) => {
    const paramSchemaExtension: Record<string, z.ZodType> = {};
    const parameters: ExtendedParamsOverviewLogs = { ...args };

    const hasIdentifierFilters = args.identifiers && args.identifiers.length > 0;
    const hasStatusFilters = args.status && args.status.length > 0;

    const statusCondition = !hasStatusFilters
      ? "TRUE"
      : args.status
          ?.map((filter, index) => {
            if (filter.operator === "is") {
              const paramName = `statusValue_${index}`;
              paramSchemaExtension[paramName] = z.boolean();
              parameters[paramName] = filter.value === "passed";
              return `passed = {${paramName}: Boolean}`;
            }
            return null;
          })
          .filter(Boolean)
          .join(" OR ") || "TRUE";

    const identifierConditions = !hasIdentifierFilters
      ? "TRUE"
      : args.identifiers
          ?.map((p, index) => {
            const paramName = `identifierValue_${index}`;
            paramSchemaExtension[paramName] = z.string();
            parameters[paramName] = p.value;
            switch (p.operator) {
              case "is":
                return `identifier = {${paramName}: String}`;
              case "contains":
                return `position({${paramName}: String}, identifier) > 0`;
              default:
                return null;
            }
          })
          .filter(Boolean)
          .join(" OR ") || "TRUE";

    const extendedParamsSchema = ratelimitOverviewLogsParams.extend(paramSchemaExtension);

    const query = ch.query({
      query: `
WITH filtered_ratelimits AS (
    SELECT
        request_id,
        time,
        identifier,
        toUInt8(passed) as status
    FROM ratelimits.raw_ratelimits_v1
    WHERE workspace_id = {workspaceId: String}
        AND namespace_id = {namespaceId: String}
        AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
        AND (${identifierConditions})
        AND (${statusCondition})
        AND (({cursorTime: Nullable(UInt64)} IS NULL AND {cursorRequestId: Nullable(String)} IS NULL)
             OR (time = {cursorTime: Nullable(UInt64)} AND request_id < {cursorRequestId: Nullable(String)})
             OR time < {cursorTime: Nullable(UInt64)})
),
filtered_metrics AS (
    SELECT 
        request_id,
        service_latency
    FROM metrics.raw_api_requests_v1
    WHERE workspace_id = {workspaceId: String}
        AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
),
aggregated_data AS (
    SELECT 
        fr.identifier,
        max(fr.time) as last_request_time,
        max(fr.request_id) as last_request_id,
        countIf(fr.status = 1) as passed_count,
        countIf(fr.status = 0) as blocked_count,
        round(avg(fm.service_latency)) as avg_latency,
        round(quantile(0.99)(fm.service_latency)) as p99_latency
    FROM filtered_ratelimits fr
    LEFT JOIN filtered_metrics fm ON fr.request_id = fm.request_id
    GROUP BY fr.identifier
)
SELECT 
    identifier,
    last_request_time as time,
    last_request_id as request_id,
    passed_count,
    blocked_count,
    avg_latency,
    p99_latency
FROM aggregated_data
ORDER BY time DESC, request_id DESC
LIMIT {limit: Int}`,
      params: extendedParamsSchema,
      schema: ratelimitOverviewLogs,
    });

    return query(parameters);
  };
}

// ## OVERVIEW Timeseries

export const ratelimitLatencyTimeseriesParams = z.object({
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

export const ratelimitLatencyTimeseriesDataPoint = z.object({
  x: dateTimeToUnix,
  y: z.object({
    avg_latency: z.number().default(0),
    p99_latency: z.number().default(0),
  }),
});

export type RatelimitLatencyTimeseriesDataPoint = z.infer<
  typeof ratelimitLatencyTimeseriesDataPoint
>;
export type RatelimitLatencyTimeseriesParams = z.infer<typeof ratelimitLatencyTimeseriesParams>;

type LatencyTimeInterval = {
  table: string;
  timeFunction: string;
  step: string;
};

const LATENCY_INTERVALS: Record<string, LatencyTimeInterval> = {
  day: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_day_v1",
    timeFunction: "toStartOfDay",
    step: "DAY",
  },
  minute: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_minute_v1",
    timeFunction: "toStartOfMinute",
    step: "MINUTE",
  },
  hour: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_hour_v1",
    timeFunction: "toStartOfHour",
    step: "HOUR",
  },
} as const;

function createLatencyTimeseriesQuery(interval: LatencyTimeInterval, whereClause: string) {
  return `
    WITH filtered_data AS (
      SELECT
        time,
        workspace_id,
        namespace_id,
        identifier,
        avg_latency,
        p99_latency
      FROM ${interval.table}
      WHERE workspace_id = {workspaceId: String}
        AND namespace_id = {namespaceId: String}
        AND time >= ${interval.timeFunction}(fromUnixTimestamp64Milli({startTime: Int64}))
        AND time <= ${interval.timeFunction}(fromUnixTimestamp64Milli({endTime: Int64}))
        ${whereClause}
    )
    SELECT
      time as x,
      map(
        'avg_latency', round(toFloat64(avgMerge(avg_latency))),
        'p99_latency', round(toFloat64(quantileMerge(0.99)(p99_latency)))
      ) as y
    FROM filtered_data
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM ${interval.timeFunction}(fromUnixTimestamp64Milli({startTime: Int64}))
      TO ${interval.timeFunction}(fromUnixTimestamp64Milli({endTime: Int64}))
      STEP INTERVAL 1 ${interval.step}
  `;
}

function getRatelimitLatencyTimeseriesWhereClause(params: RatelimitLatencyTimeseriesParams): {
  whereClause: string;
  paramSchema: z.ZodType<any>;
} {
  const conditions: string[] = [];
  const paramSchemaExtension: Record<string, z.ZodString> = {};

  if (params.identifiers?.length) {
    const identifierConditions = params.identifiers
      .map((p, index) => {
        const paramName = `identifierValue_${index}`;
        paramSchemaExtension[paramName] = z.string();
        switch (p.operator) {
          case "is":
            return `identifier = {${paramName}: String}`;
          case "contains":
            return `like(identifier, CONCAT('%', {${paramName}: String}, '%'))`;
        }
      })
      .join(" OR ");
    conditions.push(`AND (${identifierConditions})`);
  }

  return {
    whereClause: conditions.join(" "),
    paramSchema: ratelimitLatencyTimeseriesParams.extend(paramSchemaExtension),
  };
}

function createLatencyTimeseriesQuerier(interval: LatencyTimeInterval) {
  return (ch: Querier) => async (args: RatelimitLatencyTimeseriesParams) => {
    const { whereClause, paramSchema } = getRatelimitLatencyTimeseriesWhereClause(args);

    const parameters = {
      ...args,
      ...(args.identifiers?.reduce(
        (acc, i, index) => ({
          // biome-ignore lint/performance/noAccumulatingSpread: <explanation>
          ...acc,
          [`identifierValue_${index}`]: i.value,
        }),
        {},
      ) ?? {}),
    };

    return ch.query({
      query: createLatencyTimeseriesQuery(interval, whereClause),
      params: paramSchema,
      schema: ratelimitLatencyTimeseriesDataPoint,
    })(parameters);
  };
}

export const getRatelimitLatencyTimeseriesPerDay = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.day,
);
export const getRatelimitLatencyTimeseriesPerHour = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.hour,
);
export const getRatelimitLatencyTimeseriesPerMinute = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.minute,
);
