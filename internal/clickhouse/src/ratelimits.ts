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
  x: z.number().int(),
  y: z.object({
    passed: z.number().int().default(0),
    total: z.number().int().default(0),
  }),
});

export type RatelimitLogsTimeseriesDataPoint = z.infer<typeof ratelimitLogsTimeseriesDataPoint>;
export type RatelimitLogsTimeseriesParams = z.infer<typeof ratelimitLogsTimeseriesParams>;

type TimeInterval = {
  table: string;
  step: string;
  stepSize: number;
};

const INTERVALS: Record<string, TimeInterval> = {
  minute: {
    table: "ratelimits.ratelimits_per_minute_v1",
    step: "MINUTE",
    stepSize: 1,
  },
  fiveMinutes: {
    table: "ratelimits.ratelimits_per_minute_v1",
    step: "MINUTES",
    stepSize: 5,
  },
  fifteenMinutes: {
    table: "ratelimits.ratelimits_per_minute_v1",
    step: "MINUTES",
    stepSize: 15,
  },
  thirtyMinutes: {
    table: "ratelimits.ratelimits_per_minute_v1",
    step: "MINUTES",
    stepSize: 30,
  },
  hour: {
    table: "ratelimits.ratelimits_per_hour_v1",
    step: "HOUR",
    stepSize: 1,
  },
  twoHours: {
    table: "ratelimits.ratelimits_per_hour_v1",
    step: "HOURS",
    stepSize: 2,
  },
  fourHours: {
    table: "ratelimits.ratelimits_per_hour_v1",
    step: "HOURS",
    stepSize: 4,
  },
  sixHours: {
    table: "ratelimits.ratelimits_per_hour_v1",
    step: "HOURS",
    stepSize: 6,
  },
  day: {
    table: "ratelimits.ratelimits_per_day_v1",
    step: "DAY",
    stepSize: 1,
  },
  month: {
    table: "ratelimits.ratelimits_per_month_v1",
    step: "MONTH",
    stepSize: 1,
  },
} as const;

function createTimeseriesQuery(interval: TimeInterval, whereClause: string) {
  const intervalUnit = {
    MINUTE: 60_000, // milliseconds in a minute
    MINUTES: 60_000,
    HOUR: 3600_000, // milliseconds in an hour
    HOURS: 3600_000,
    DAY: 86400_000, // milliseconds in a day
    MONTH: 2592000_000, // approximate milliseconds in a month (30 days)
  }[interval.step];

  const stepMs = intervalUnit! * interval.stepSize;

  return `
    SELECT
      toUnixTimestamp64Milli(CAST(toStartOfInterval(time, INTERVAL ${interval.stepSize} ${interval.step}) AS DateTime64(3))) as x,
      map(
        'passed', sum(passed),
        'total', sum(total)
      ) as y
    FROM ${interval.table}
    ${whereClause}
    GROUP BY x
    ORDER BY x ASC
    WITH FILL
      FROM toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({startTime: Int64})), INTERVAL ${interval.stepSize} ${interval.step}) AS DateTime64(3)))
      TO toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({endTime: Int64})), INTERVAL ${interval.stepSize} ${interval.step}) AS DateTime64(3)))
      STEP ${stepMs}`;
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

export const getMinutelyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.minute);
export const getFiveMinuteRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.fiveMinutes);
export const getFifteenMinuteRatelimitTimeseries = createTimeseriesQuerier(
  INTERVALS.fifteenMinutes,
);
export const getThirtyMinuteRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.thirtyMinutes);
export const getHourlyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.hour);
export const getTwoHourlyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.twoHours);
export const getFourHourlyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.fourHours);
export const getSixHourlyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.sixHours);
export const getDailyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.day);
export const getMonthlyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.month);

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
                return `like(identifier, CONCAT('%', {${paramName}: String}, '%'))`;
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

  sorts: z
    .array(
      z.object({
        column: z.enum(["time", "avg_latency", "p99_latency", "blocked", "passed"]),
        direction: z.enum(["asc", "desc"]),
      }),
    )
    .nullable(),
});

export const ratelimitOverviewLogs = z.object({
  time: z.number().int(),
  identifier: z.string(),
  request_id: z.string(),
  passed_count: z.number().int(),
  blocked_count: z.number().int(),
  // avg_latency: z.number().int(),
  // p99_latency: z.number().int(),
  override: z
    .object({
      limit: z.number().int(),
      duration: z.number().int(),
      overrideId: z.string(),
      async: z.boolean().nullable(),
    })
    .optional()
    .nullable(),
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
    const hasSortingRules = args.sorts && args.sorts.length > 0;

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
                return `like(identifier, CONCAT('%', {${paramName}: String}, '%'))`;
              default:
                return null;
            }
          })
          .filter(Boolean)
          .join(" OR ") || "TRUE";

    const allowedColumns = new Map([
      ["time", "last_request_time"],
      ["avg_latency", "avg_latency"],
      ["p99_latency", "p99_latency"],
      ["passed", "passed_count"],
      ["blocked", "blocked_count"],
    ]);

    const orderBy =
      hasSortingRules && args.sorts
        ? args.sorts.reduce((acc: string[], sort) => {
            const column = allowedColumns.get(sort.column);
            // Only add to ORDER BY if it's an allowed column to prevent injection
            if (column) {
              const direction =
                sort.direction.toUpperCase() === "ASC" || sort.direction.toUpperCase() === "DESC"
                  ? sort.direction.toUpperCase()
                  : "DESC";
              acc.push(`${column} ${direction}`);
            }
            return acc;
          }, [])
        : [];

    // Check if we have custom sorts
    const hasAvgLatencySort = args.sorts?.some((s) => s.column === "avg_latency");
    const hasP99LatencySort = args.sorts?.some((s) => s.column === "p99_latency");
    const hasPassedSort = args.sorts?.some((s) => s.column === "passed");
    const hasBlockedSort = args.sorts?.some((s) => s.column === "blocked");
    const hasCustomSort = hasAvgLatencySort || hasP99LatencySort || hasPassedSort || hasBlockedSort;

    // Get explicit time sort if it exists
    const timeSort = args.sorts?.find((s) => s.column === "time");

    // If we have custom sort (avg_latency, p99_latency, passed, blocked), always use ASC for better pagination
    // Otherwise use explicit time direction or default to DESC
    const timeDirection = hasCustomSort
      ? "ASC"
      : timeSort?.direction.toUpperCase() === "ASC"
        ? "ASC"
        : "DESC";

    // Remove any existing time sort from the orderBy array
    const orderByWithoutTime = orderBy.filter((clause) => !clause.startsWith("last_request_time"));

    // Construct final ORDER BY clause with time and request_id always at the end
    const orderByClause =
      [
        ...orderByWithoutTime,
        `last_request_time ${timeDirection}`,
        `request_id ${timeDirection}`,
      ].join(", ") || "last_request_time DESC, request_id DESC"; // Fallback if empty

    // Create cursor condition based on time direction
    let cursorCondition: string;

    // For first page or no cursor provided
    if (!args.cursorTime || !args.cursorRequestId) {
      cursorCondition = `
      AND ({cursorTime: Nullable(UInt64)} IS NULL AND {cursorRequestId: Nullable(String)} IS NULL)
      `;
    } else {
      // For subsequent pages, use cursor based on time direction
      if (timeDirection === "ASC") {
        cursorCondition = `
        AND (
            (time = {cursorTime: Nullable(UInt64)} AND request_id > {cursorRequestId: Nullable(String)})
            OR time > {cursorTime: Nullable(UInt64)}
        )
        `;
      } else {
        cursorCondition = `
        AND (
            (time = {cursorTime: Nullable(UInt64)} AND request_id < {cursorRequestId: Nullable(String)})
            OR time < {cursorTime: Nullable(UInt64)}
        )
        `;
      }
    }

    const extendedParamsSchema = ratelimitOverviewLogsParams.extend(paramSchemaExtension);
    const query = ch.query({
      query: `WITH filtered_ratelimits AS (
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
        ${cursorCondition}
),
aggregated_data AS (
    SELECT 
        identifier,
        max(time) as last_request_time,
        max(request_id) as last_request_id,
        countIf(status = 1) as passed_count,
        countIf(status = 0) as blocked_count
    FROM filtered_ratelimits
    GROUP BY identifier
)
SELECT 
    identifier,
    last_request_time as time,
    last_request_id as request_id,
    passed_count,
    blocked_count
FROM aggregated_data
ORDER BY ${orderByClause}
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

const LATENCY_INTERVALS: Record<string, TimeInterval> = {
  minute: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_minute_v1",
    step: "MINUTE",
    stepSize: 1,
  },
  fiveMinutes: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_minute_v1",
    step: "MINUTES",
    stepSize: 5,
  },
  fifteenMinutes: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_minute_v1",
    step: "MINUTES",
    stepSize: 15,
  },
  thirtyMinutes: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_minute_v1",
    step: "MINUTES",
    stepSize: 30,
  },
  hour: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_hour_v1",
    step: "HOUR",
    stepSize: 1,
  },
  twoHours: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_hour_v1",
    step: "HOURS",
    stepSize: 2,
  },
  fourHours: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_hour_v1",
    step: "HOURS",
    stepSize: 4,
  },
  sixHours: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_hour_v1",
    step: "HOURS",
    stepSize: 6,
  },
  day: {
    table: "ratelimits.ratelimits_identifier_latency_stats_per_day_v1",
    step: "DAY",
    stepSize: 1,
  },
} as const;

function createLatencyTimeseriesQuery(interval: TimeInterval, whereClause: string) {
  // Map step to ClickHouse interval unit
  const intervalUnit = {
    MINUTE: "minute",
    MINUTES: "minute",
    HOUR: "hour",
    HOURS: "hour",
    DAY: "day",
  }[interval.step];

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
        AND time >= toStartOfInterval(fromUnixTimestamp64Milli({startTime: Int64}), INTERVAL ${interval.stepSize} ${intervalUnit})
        AND time <= toStartOfInterval(fromUnixTimestamp64Milli({endTime: Int64}), INTERVAL ${interval.stepSize} ${intervalUnit})
        ${whereClause}
    )
    SELECT
      toStartOfInterval(time, INTERVAL ${interval.stepSize} ${intervalUnit}) as x,
      map(
        'avg_latency', round(toFloat64(avgMerge(avg_latency))),
        'p99_latency', round(toFloat64(quantileMerge(0.99)(p99_latency)))
      ) as y
    FROM filtered_data
    GROUP BY x
    ORDER BY x ASC
    WITH FILL
      FROM toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({startTime: Int64})), INTERVAL ${interval.stepSize} ${intervalUnit})
      TO toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({endTime: Int64})), INTERVAL ${interval.stepSize} ${intervalUnit})
      STEP INTERVAL ${interval.stepSize} ${intervalUnit}
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

function createLatencyTimeseriesQuerier(interval: TimeInterval) {
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

export const getMinutelyLatencyTimeseries = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.minute,
);
export const getFiveMinuteLatencyTimeseries = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.fiveMinutes,
);
export const getFifteenMinuteLatencyTimeseries = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.fifteenMinutes,
);
export const getThirtyMinuteLatencyTimeseries = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.thirtyMinutes,
);
export const getHourlyLatencyTimeseries = createLatencyTimeseriesQuerier(LATENCY_INTERVALS.hour);
export const getTwoHourlyLatencyTimeseries = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.twoHours,
);
export const getFourHourlyLatencyTimeseries = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.fourHours,
);
export const getSixHourlyLatencyTimeseries = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.sixHours,
);
export const getDailyLatencyTimeseries = createLatencyTimeseriesQuerier(LATENCY_INTERVALS.day);
