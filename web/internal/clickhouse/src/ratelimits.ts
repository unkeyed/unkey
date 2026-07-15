import { z } from "zod";
import type { Inserter, Querier } from "./client";
import { dateTimeToUnix } from "./util";

export function insertRatelimit(ch: Inserter) {
  return ch.insert({
    table: "ratelimits.raw_ratelimits_v1",
    schema: z.object({
      request_id: z.string(),
      time: z.int(),
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
  startTime: z.int(),
  endTime: z.int(),
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
  x: z.int(),
  y: z.object({
    passed: z.int().prefault(0),
    total: z.int().prefault(0),
    // total_tokens = sum of tokens across all decisions in the bucket.
    // passed_tokens = sum of tokens for decisions where passed=true.
    // Blocked tokens are derived in the UI as total_tokens - passed_tokens.
    passed_tokens: z.int().prefault(0),
    total_tokens: z.int().prefault(0),
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
    table: "default.ratelimits_per_minute_v2",
    step: "MINUTE",
    stepSize: 1,
  },
  fiveMinutes: {
    table: "default.ratelimits_per_minute_v2",
    step: "MINUTES",
    stepSize: 5,
  },
  fifteenMinutes: {
    table: "default.ratelimits_per_minute_v2",
    step: "MINUTES",
    stepSize: 15,
  },
  thirtyMinutes: {
    table: "default.ratelimits_per_minute_v2",
    step: "MINUTES",
    stepSize: 30,
  },
  hour: {
    table: "default.ratelimits_per_hour_v2",
    step: "HOUR",
    stepSize: 1,
  },
  twoHours: {
    table: "default.ratelimits_per_hour_v2",
    step: "HOURS",
    stepSize: 2,
  },
  fourHours: {
    table: "default.ratelimits_per_hour_v2",
    step: "HOURS",
    stepSize: 4,
  },
  sixHours: {
    table: "default.ratelimits_per_hour_v2",
    step: "HOURS",
    stepSize: 6,
  },
  twelveHours: {
    table: "default.ratelimits_per_hour_v2",
    step: "HOUR",
    stepSize: 12,
  },
  day: {
    table: "default.ratelimits_per_day_v2",
    step: "DAY",
    stepSize: 1,
  },
  threeDays: {
    table: "default.ratelimits_per_day_v2",
    step: "DAY",
    stepSize: 3,
  },
  week: {
    table: "default.ratelimits_per_day_v2",
    step: "DAY",
    stepSize: 7,
  },
  month: {
    table: "default.ratelimits_per_month_v2",
    step: "MONTH",
    stepSize: 1,
  },
  quarter: {
    table: "default.ratelimits_per_month_v2",
    step: "MONTH",
    stepSize: 3,
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

  if (!intervalUnit) {
    throw new Error("Unknown interval in 'createTimeseriesQuery'");
  }

  const stepMs = intervalUnit * interval.stepSize;

  return `
    SELECT
      toUnixTimestamp64Milli(CAST(toStartOfInterval(time, INTERVAL ${interval.stepSize} ${interval.step}) AS DateTime64(3))) as x,
      map(
        'passed', sum(passed),
        'total', sum(total),
        'passed_tokens', sum(passed_tokens),
        'total_tokens', sum(total_tokens)
      ) as y
    FROM ${interval.table}
    ${whereClause}
    GROUP BY x
    ORDER BY x ASC
    WITH FILL
      FROM toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({startTime: Int64})), INTERVAL ${interval.stepSize} ${interval.step}) AS DateTime64(3)))
      TO toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({endTime: Int64})), INTERVAL ${interval.stepSize} ${interval.step}) AS DateTime64(3))) + ${stepMs}
      STEP ${stepMs}`;
}

function getRatelimitLogsTimeseriesWhereClause(
  params: RatelimitLogsTimeseriesParams,
  additionalConditions: string[] = [],
): { whereClause: string; paramSchema: z.ZodType<unknown> } {
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

// Batch timeseries: query multiple namespaces in a single ClickHouse call
export const ratelimitBatchTimeseriesParams = z.object({
  workspaceId: z.string(),
  namespaceIds: z.array(z.string()),
  startTime: z.int(),
  endTime: z.int(),
});

export const ratelimitBatchTimeseriesDataPoint = z.object({
  namespace_id: z.string(),
  x: z.int(),
  y: z.object({
    passed: z.int().prefault(0),
    total: z.int().prefault(0),
  }),
});

export type RatelimitBatchTimeseriesParams = z.infer<typeof ratelimitBatchTimeseriesParams>;
export type RatelimitBatchTimeseriesDataPoint = z.infer<typeof ratelimitBatchTimeseriesDataPoint>;

function createBatchTimeseriesQuery(interval: TimeInterval) {
  const intervalUnit = {
    MINUTE: 60_000,
    MINUTES: 60_000,
    HOUR: 3600_000,
    HOURS: 3600_000,
    DAY: 86400_000,
    MONTH: 2592000_000,
  }[interval.step];

  if (!intervalUnit) {
    throw new Error("Unknown interval in 'createBatchTimeseriesQuery'");
  }

  return `
    SELECT
      namespace_id,
      toUnixTimestamp64Milli(CAST(toStartOfInterval(time, INTERVAL ${interval.stepSize} ${interval.step}) AS DateTime64(3))) as x,
      map(
        'passed', sum(passed),
        'total', sum(total)
      ) as y
    FROM ${interval.table}
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id IN {namespaceIds: Array(String)}
      AND time >= fromUnixTimestamp64Milli({startTime: Int64})
      AND time <= fromUnixTimestamp64Milli({endTime: Int64})
    GROUP BY namespace_id, x
    ORDER BY namespace_id, x ASC`;
}

function createBatchTimeseriesQuerier(interval: TimeInterval) {
  return (ch: Querier) => async (args: RatelimitBatchTimeseriesParams) => {
    return ch.query({
      query: createBatchTimeseriesQuery(interval),
      params: ratelimitBatchTimeseriesParams,
      schema: ratelimitBatchTimeseriesDataPoint,
    })(args);
  };
}

export const getBatchMinutelyRatelimitTimeseries = createBatchTimeseriesQuerier(INTERVALS.minute);
export const getBatchFiveMinuteRatelimitTimeseries = createBatchTimeseriesQuerier(
  INTERVALS.fiveMinutes,
);
export const getBatchFifteenMinuteRatelimitTimeseries = createBatchTimeseriesQuerier(
  INTERVALS.fifteenMinutes,
);
export const getBatchThirtyMinuteRatelimitTimeseries = createBatchTimeseriesQuerier(
  INTERVALS.thirtyMinutes,
);
export const getBatchHourlyRatelimitTimeseries = createBatchTimeseriesQuerier(INTERVALS.hour);
export const getBatchTwoHourlyRatelimitTimeseries = createBatchTimeseriesQuerier(
  INTERVALS.twoHours,
);
export const getBatchFourHourlyRatelimitTimeseries = createBatchTimeseriesQuerier(
  INTERVALS.fourHours,
);
export const getBatchSixHourlyRatelimitTimeseries = createBatchTimeseriesQuerier(
  INTERVALS.sixHours,
);
export const getBatchTwelveHourlyRatelimitTimeseries = createBatchTimeseriesQuerier(
  INTERVALS.twelveHours,
);
export const getBatchDailyRatelimitTimeseries = createBatchTimeseriesQuerier(INTERVALS.day);
export const getBatchThreeDayRatelimitTimeseries = createBatchTimeseriesQuerier(
  INTERVALS.threeDays,
);
export const getBatchWeeklyRatelimitTimeseries = createBatchTimeseriesQuerier(INTERVALS.week);
export const getBatchMonthlyRatelimitTimeseries = createBatchTimeseriesQuerier(INTERVALS.month);
export const getBatchQuarterlyRatelimitTimeseries = createBatchTimeseriesQuerier(INTERVALS.quarter);

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
export const getTwelveHourlyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.twelveHours);
export const getDailyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.day);
export const getThreeDayRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.threeDays);
export const getWeeklyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.week);
export const getMonthlyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.month);
export const getQuarterlyRatelimitTimeseries = createTimeseriesQuerier(INTERVALS.quarter);

const getRatelimitLastUsedParameters = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  identifier: z.array(z.string()).optional(),
  limit: z.int(),
});

export function getRatelimitLastUsed(ch: Querier) {
  return async (args: z.input<typeof getRatelimitLastUsedParameters>) => {
    const query = ch.query({
      query: `
    SELECT
      identifier,
      max(time) as time
    FROM default.ratelimits_last_used_v2
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

// Columns the logs list can be ordered by. These map to physical
// `ratelimits_raw_v2` columns (see SORT_COLUMN_SQL), so the set is closed —
// never derive an ORDER BY column from a raw client string.
export const ratelimitLogsSort = z.object({
  column: z.enum(["time", "identifier", "status"]),
  direction: z.enum(["asc", "desc"]),
});

export type RatelimitLogsSort = z.infer<typeof ratelimitLogsSort>;

export const ratelimitLogsParams = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  limit: z.int(),
  startTime: z.int(),
  endTime: z.int(),
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
  sorts: z.array(ratelimitLogsSort).nullable(),
  offset: z.int(),
});

export const ratelimitLogs = z.object({
  request_id: z.string(),
  time: z.int(),
  identifier: z.string(),
  status: z.int(),
});

export const ratelimitLogEnrichment = z.object({
  request_id: z.string(),
  host: z.string(),
  method: z.string(),
  path: z.string(),
  request_headers: z.array(z.string()),
  request_body: z.string(),
  response_status: z.int(),
  response_headers: z.array(z.string()),
  response_body: z.string(),
  service_latency: z.int(),
  user_agent: z.string(),
  region: z.string(),
});

export type RatelimitLog = z.infer<typeof ratelimitLogs>;
export type RatelimitLogEnrichment = z.infer<typeof ratelimitLogEnrichment>;
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

    const statusCondition = hasStatusFilters
      ? args.status
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
          .join(" OR ") || "TRUE"
      : "TRUE";

    const identifierConditions = hasIdentifierFilters
      ? args.identifiers
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
          .join(" OR ") || "TRUE"
      : "TRUE";

    const extendedParamsSchema = ratelimitLogsParams.extend(paramSchemaExtension);

    // ORDER BY is built from a closed column allowlist keyed by the validated
    // `sorts` enum — column names and directions are never raw client strings,
    // so this cannot be an injection vector. `time DESC` is always appended as
    // the final tiebreaker so OFFSET pagination stays deterministic when
    // sorting by a non-unique column (identifier/status).
    const SORT_COLUMN_SQL: Record<RatelimitLogsSort["column"], string> = {
      time: "time",
      identifier: "identifier",
      status: "passed",
    };
    const sorts = args.sorts ?? [];
    const orderByParts = sorts.map(
      (sort) => `${SORT_COLUMN_SQL[sort.column]} ${sort.direction === "asc" ? "ASC" : "DESC"}`,
    );
    if (!sorts.some((sort) => sort.column === "time")) {
      orderByParts.push("time DESC");
    }
    const orderByClause = orderByParts.join(", ");

    const logsQuery = ch.query({
      query: `
SELECT
    request_id,
    time,
    identifier,
    toUInt8(passed) as status
FROM default.ratelimits_raw_v2
WHERE workspace_id = {workspaceId: String}
    AND namespace_id = {namespaceId: String}
    AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
    ${hasRequestIds ? "AND request_id IN {requestIds: Array(String)}" : ""}
    AND (${identifierConditions})
    AND (${statusCondition})
ORDER BY ${orderByClause}
LIMIT {limit: Int}
OFFSET {offset: Int}`,
      params: extendedParamsSchema,
      schema: ratelimitLogs,
    });

    const countQuery = ch.query({
      query: `
SELECT
    count(*) as total_count
FROM default.ratelimits_raw_v2 r
WHERE workspace_id = {workspaceId: String}
    AND namespace_id = {namespaceId: String}
    AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
    ${hasRequestIds ? "AND request_id IN {requestIds: Array(String)}" : ""}
    AND (${identifierConditions})
    AND (${statusCondition})`,
      params: extendedParamsSchema,
      schema: z.object({
        total_count: z.int(),
      }),
    });

    return {
      logsQuery: logsQuery(parameters),
      countQuery: countQuery(parameters),
    };
  };
}

export const ratelimitLogEnrichmentParams = z.object({
  workspaceId: z.string(),
  requestIds: z.array(z.string()),
  startTime: z.int(),
  endTime: z.int(),
});

export type RatelimitLogEnrichmentParams = z.infer<typeof ratelimitLogEnrichmentParams>;

export function getRatelimitLogEnrichment(ch: Querier) {
  return async (args: RatelimitLogEnrichmentParams) => {
    const query = ch.query({
      query: `
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
    region
FROM default.api_requests_raw_v2
WHERE workspace_id = {workspaceId: String}
    AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
    AND request_id IN {requestIds: Array(String)}`,
      params: ratelimitLogEnrichmentParams,
      schema: ratelimitLogEnrichment,
    });

    return query(args);
  };
}

export const ratelimitOverviewLogsParams = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  limit: z.int(),
  startTime: z.int(),
  endTime: z.int(),
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
  page: z.int().min(1).optional(),
  sorts: z
    .array(
      z.object({
        column: z.enum(["time", "blocked", "passed", "passed_tokens", "blocked_tokens"]),
        direction: z.enum(["asc", "desc"]),
      }),
    )
    .nullable(),
});

export const ratelimitOverviewLogs = z.object({
  time: z.int(),
  identifier: z.string(),
  request_id: z.string(),
  passed_count: z.int(),
  blocked_count: z.int(),
  // total_tokens = sum of tokens across all decisions for the identifier in
  // the window. passed_tokens = sum of tokens for decisions where passed=true.
  // Blocked tokens are derived in the UI as total_tokens - passed_tokens so
  // the table reconciles with the bar chart's blocked-tokens series.
  passed_tokens: z.int().prefault(0),
  total_tokens: z.int().prefault(0),
  override: z
    .object({
      limit: z.int(),
      duration: z.int(),
      overrideId: z.string(),
    })
    .optional()
    .nullable(),
});

export type RatelimitOverviewLog = z.infer<typeof ratelimitOverviewLogs>;
export type RatelimitOverviewLogsParams = z.infer<typeof ratelimitOverviewLogsParams>;

interface ExtendedParamsOverviewLogs extends RatelimitOverviewLogsParams {
  [key: string]: unknown;
}

// getRatelimitOverviewLogs returns per-identifier rate limit aggregates for a
// namespace, paginated with LIMIT/OFFSET over the `page` argument. Because
// OFFSET is only stable under a total ordering, the ORDER BY always ends with
// `last_request_time` then `request_id` as tiebreakers so a row never appears
// on two pages or gets skipped between them. The total row count rides along on
// each page via `count() OVER ()`; only when a page lands past the end (no rows)
// does it fall back to a dedicated count query so the caller can clamp the page.
export function getRatelimitOverviewLogs(ch: Querier) {
  return async (args: RatelimitOverviewLogsParams) => {
    const paramSchemaExtension: Record<string, z.ZodType> = {};
    const parameters: ExtendedParamsOverviewLogs = { ...args };

    const hasIdentifierFilters = args.identifiers && args.identifiers.length > 0;
    const hasStatusFilters = args.status && args.status.length > 0;
    const hasSortingRules = args.sorts && args.sorts.length > 0;

    const statusCondition = hasStatusFilters
      ? args.status
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
          .join(" OR ") || "TRUE"
      : "TRUE";

    const identifierConditions = hasIdentifierFilters
      ? args.identifiers
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
          .join(" OR ") || "TRUE"
      : "TRUE";

    const allowedColumns = new Map([
      ["time", "last_request_time"],
      ["passed", "passed_count"],
      ["blocked", "blocked_count"],
      ["passed_tokens", "passed_tokens"],
      ["blocked_tokens", "blocked_tokens"],
    ]);

    const toSqlDirection = (direction: "asc" | "desc"): "ASC" | "DESC" =>
      direction === "asc" ? "ASC" : "DESC";

    const orderBy =
      hasSortingRules && args.sorts
        ? args.sorts.reduce((acc: string[], sort) => {
            const column = allowedColumns.get(sort.column);
            if (column) {
              acc.push(`${column} ${toSqlDirection(sort.direction)}`);
            }
            return acc;
          }, [])
        : [];

    // When sorting by a non-time column, time falls through to a stable
    // tiebreaker (ASC) so OFFSET pagination stays deterministic between pages.
    const hasNonTimeSort = args.sorts?.some((s) => s.column !== "time") ?? false;
    const timeSort = args.sorts?.find((s) => s.column === "time");
    const timeDirection: "ASC" | "DESC" = hasNonTimeSort
      ? "ASC"
      : timeSort
        ? toSqlDirection(timeSort.direction)
        : "DESC";

    const orderByWithoutTime = orderBy.filter((clause) => !clause.startsWith("last_request_time"));

    const orderByClause = [
      ...orderByWithoutTime,
      `last_request_time ${timeDirection}`,
      `request_id ${timeDirection}`,
    ].join(", ");

    const page = args.page ?? 1;
    const offset = (page - 1) * args.limit;
    parameters.offset = offset;
    paramSchemaExtension.offset = z.int();

    const extendedParamsSchema = ratelimitOverviewLogsParams.extend(paramSchemaExtension);
    const query = ch.query({
      query: `WITH filtered_ratelimits AS (
    SELECT
        request_id,
        time,
        identifier,
        toUInt8(passed) as status,
        tokens
    FROM default.ratelimits_raw_v2
    WHERE workspace_id = {workspaceId: String}
        AND namespace_id = {namespaceId: String}
        AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
        AND (${identifierConditions})
        AND (${statusCondition})
),
aggregated_data AS (
    SELECT
        identifier,
        max(time) as last_request_time,
        max(request_id) as last_request_id,
        countIf(status = 1) as passed_count,
        countIf(status = 0) as blocked_count,
        sumIf(tokens, status = 1) as passed_tokens,
        sum(tokens) as total_tokens,
        greatest(sum(tokens) - sumIf(tokens, status = 1), 0) as blocked_tokens
    FROM filtered_ratelimits
    GROUP BY identifier
)
SELECT
    identifier,
    last_request_time as time,
    last_request_id as request_id,
    passed_count,
    blocked_count,
    passed_tokens,
    total_tokens,
    count() OVER () as total_count
FROM aggregated_data
ORDER BY ${orderByClause}
LIMIT {limit: Int}
OFFSET {offset: Int}`,
      params: extendedParamsSchema,
      schema: ratelimitOverviewLogs.extend({
        total_count: z.int(),
      }),
    });

    const countOnlyQuery = ch.query({
      query: `
SELECT
    count(DISTINCT identifier) as total_count
FROM default.ratelimits_raw_v2
WHERE workspace_id = {workspaceId: String}
    AND namespace_id = {namespaceId: String}
    AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
    AND (${identifierConditions})
    AND (${statusCondition})`,
      params: extendedParamsSchema,
      schema: z.object({
        total_count: z.int(),
      }),
    });

    const sharedResult = query(parameters);

    const logsQuery = (async () => {
      const result = await sharedResult;
      if (result.err) {
        return result;
      }
      return {
        err: result.err,
        val: (result.val ?? []).map((row) => ({
          time: row.time,
          identifier: row.identifier,
          request_id: row.request_id,
          passed_count: row.passed_count,
          blocked_count: row.blocked_count,
          passed_tokens: row.passed_tokens,
          total_tokens: row.total_tokens,
        })),
      };
    })();

    // Common case (paged result has rows): total ships per row via
    // `count() OVER ()`. Edge case (page past end / no matches): the window
    // function emits no rows, so fall through to a dedicated count query so
    // the UI can clamp the page.
    const countQuery = (async () => {
      const result = await sharedResult;
      if (result.err) {
        return { err: result.err, val: undefined };
      }
      if (result.val && result.val.length > 0) {
        return { err: undefined, val: [{ total_count: result.val[0].total_count }] };
      }
      return countOnlyQuery(parameters);
    })();

    return {
      logsQuery,
      countQuery,
    };
  };
}

// ## OVERVIEW Timeseries
export const ratelimitLatencyTimeseriesParams = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  startTime: z.int(),
  endTime: z.int(),
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
    avg_latency: z.number().prefault(0),
    p99_latency: z.number().prefault(0),
  }),
});

export type RatelimitLatencyTimeseriesDataPoint = z.infer<
  typeof ratelimitLatencyTimeseriesDataPoint
>;
export type RatelimitLatencyTimeseriesParams = z.infer<typeof ratelimitLatencyTimeseriesParams>;

const LATENCY_INTERVALS: Record<string, TimeInterval> = {
  minute: {
    table: "default.ratelimits_identifier_latency_stats_per_minute_v2",
    step: "MINUTE",
    stepSize: 1,
  },
  fiveMinutes: {
    table: "default.ratelimits_identifier_latency_stats_per_minute_v2",
    step: "MINUTES",
    stepSize: 5,
  },
  fifteenMinutes: {
    table: "default.ratelimits_identifier_latency_stats_per_minute_v2",
    step: "MINUTES",
    stepSize: 15,
  },
  thirtyMinutes: {
    table: "default.ratelimits_identifier_latency_stats_per_minute_v2",
    step: "MINUTES",
    stepSize: 30,
  },
  hour: {
    table: "default.ratelimits_identifier_latency_stats_per_hour_v2",
    step: "HOUR",
    stepSize: 1,
  },
  twoHours: {
    table: "default.ratelimits_identifier_latency_stats_per_hour_v2",
    step: "HOURS",
    stepSize: 2,
  },
  fourHours: {
    table: "default.ratelimits_identifier_latency_stats_per_hour_v2",
    step: "HOURS",
    stepSize: 4,
  },
  sixHours: {
    table: "default.ratelimits_identifier_latency_stats_per_hour_v2",
    step: "HOURS",
    stepSize: 6,
  },
  day: {
    table: "default.ratelimits_identifier_latency_stats_per_day_v2",
    step: "DAY",
    stepSize: 1,
  },
  threeDays: {
    table: "default.ratelimits_identifier_latency_stats_per_day_v2",
    step: "DAY",
    stepSize: 3,
  },
  week: {
    table: "default.ratelimits_identifier_latency_stats_per_week_v2",
    step: "WEEK",
    stepSize: 1,
  },
  month: {
    table: "default.ratelimits_identifier_latency_stats_per_month_v2",
    step: "MONTH",
    stepSize: 1,
  },
  threeMonths: {
    table: "default.ratelimits_identifier_latency_stats_per_month_v2",
    step: "MONTH",
    stepSize: 3,
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
  paramSchema: z.ZodType<unknown>;
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
export const getTwelveHourlyLatencyTimeseries = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.twelveHours,
);
export const getDailyLatencyTimeseries = createLatencyTimeseriesQuerier(LATENCY_INTERVALS.day);
export const getThreeDayLatencyTimeseries = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.threeDays,
);
export const getWeeklyLatencyTimeseries = createLatencyTimeseriesQuerier(LATENCY_INTERVALS.week);
export const getMonthlyLatencyTimeseries = createLatencyTimeseriesQuerier(LATENCY_INTERVALS.month);
export const getQuarterlyLatencyTimeseries = createLatencyTimeseriesQuerier(
  LATENCY_INTERVALS.quarter,
);
