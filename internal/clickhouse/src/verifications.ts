import { z } from "zod";
import type { Querier } from "./client";
import { KEY_VERIFICATION_OUTCOMES } from "./keys/keys";

// LOGS
export const keyDetailsLogsParams = z.object({
  workspaceId: z.string(),
  keyspaceId: z.string(),
  keyId: z.string(),
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  tags: z
    .array(
      z.object({
        value: z.string(),
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
      }),
    )
    .nullable(),
  outcomes: z
    .array(
      z.object({
        value: z.enum(KEY_VERIFICATION_OUTCOMES),
        operator: z.literal("is"),
      }),
    )
    .nullable(),
  cursorTime: z.number().int().nullable(),
});

export const keyDetailsLog = z.object({
  request_id: z.string(),
  time: z.number().int(),
  region: z.string(),
  outcome: z.enum(KEY_VERIFICATION_OUTCOMES),
  tags: z.array(z.string()),
});

export type KeyDetailsLog = z.infer<typeof keyDetailsLog>;
export type KeyDetailsLogsParams = z.infer<typeof keyDetailsLogsParams>;

type ExtendedParamsKeyDetails = KeyDetailsLogsParams & {
  [key: string]: unknown;
};

export function getKeyDetailsLogs(ch: Querier) {
  return async (args: KeyDetailsLogsParams) => {
    const paramSchemaExtension: Record<string, z.ZodType> = {};
    const parameters: ExtendedParamsKeyDetails = { ...args };

    const hasTagFilters = args.tags && args.tags.length > 0;
    const hasOutcomeFilters = args.outcomes && args.outcomes.length > 0;

    const tagCondition = hasTagFilters
      ? args.tags
          ?.map((filter, index) => {
            const paramName = `tagValue_${index}`;
            paramSchemaExtension[paramName] = z.string();
            parameters[paramName] = filter.value;

            switch (filter.operator) {
              case "is":
                return `has(tags, {${paramName}: String})`;
              case "contains":
                return `arrayExists(tag -> position(tag, {${paramName}: String}) > 0, tags)`;
              case "startsWith":
                return `arrayExists(tag -> startsWith(tag, {${paramName}: String}), tags)`;
              case "endsWith":
                return `arrayExists(tag -> endsWith(tag, {${paramName}: String}), tags)`;
              default:
                return null;
            }
          })
          .filter(Boolean)
          .join(" AND ") || "TRUE"
      : "TRUE";

    const outcomeCondition = hasOutcomeFilters
      ? args.outcomes
          ?.map((filter, index) => {
            if (filter.operator === "is") {
              const paramName = `outcomeValue_${index}`;
              paramSchemaExtension[paramName] = z.string();
              parameters[paramName] = filter.value;
              return `outcome = {${paramName}: String}`;
            }
            return null;
          })
          .filter(Boolean)
          .join(" OR ") || "TRUE"
      : "TRUE";

    let cursorCondition: string;

    // For first page or no cursor provided
    if (args.cursorTime) {
      cursorCondition = `
        AND (time < {cursorTime: Nullable(UInt64)})
        `;
    } else {
      cursorCondition = `
      AND ({cursorTime: Nullable(UInt64)} IS NULL)
      `;
    }

    const extendedParamsSchema = keyDetailsLogsParams.extend(paramSchemaExtension);

    const baseConditions = `
      workspace_id = {workspaceId: String}
      AND key_space_id = {keyspaceId: String}
      AND key_id = {keyId: String}
      AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
      AND (${tagCondition})
      AND (${outcomeCondition})
    `;

    // Total count query - counts all matching records without pagination
    const totalQuery = ch.query({
      query: `
        SELECT
          count(request_id) as total_count
        FROM default.key_verifications_raw_v2
        WHERE ${baseConditions}`,
      params: extendedParamsSchema,
      schema: z.object({
        total_count: z.number().int(),
      }),
    });

    const query = ch.query({
      query: `
      SELECT
          request_id,
          time,
          region,
          outcome,
          tags
      FROM default.key_verifications_raw_v2
      WHERE ${baseConditions}
          -- Handle pagination using time as cursor
          ${cursorCondition}
      ORDER BY time DESC
      LIMIT {limit: Int}
      `,
      params: extendedParamsSchema,
      schema: keyDetailsLog,
    });

    const [clickhouseResults, totalResults] = await Promise.all([
      query(parameters),
      totalQuery(parameters),
    ]);

    return {
      logs: clickhouseResults,
      totalCount: totalResults.val ? totalResults.val[0].total_count : 0,
    };
  };
}

// TIMESERIES
export const verificationTimeseriesParams = z.object({
  workspaceId: z.string(),
  keyspaceId: z.string(),
  keyId: z.string().optional(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  names: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      }),
    )
    .nullable(),
  identities: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      }),
    )
    .nullable(),
  keyIds: z
    .array(
      z.object({
        operator: z.enum(["is", "contains"]),
        value: z.string(),
      }),
    )
    .nullable(),
  tags: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      }),
    )
    .nullable(),
  outcomes: z
    .array(
      z.object({
        value: z.enum(KEY_VERIFICATION_OUTCOMES),
        operator: z.literal("is"),
      }),
    )
    .nullable(),
});

export const verificationTimeseriesDataPoint = z.object({
  x: z.number().int(),
  y: z.object({
    total: z.number().int().default(0),
    valid: z.number().int().default(0),
    valid_count: z.number().int().default(0),
    rate_limited_count: z.number().int().default(0),
    insufficient_permissions_count: z.number().int().default(0),
    forbidden_count: z.number().int().default(0),
    disabled_count: z.number().int().default(0),
    expired_count: z.number().int().default(0),
    usage_exceeded_count: z.number().int().default(0),
  }),
});

export type VerificationTimeseriesDataPoint = z.infer<typeof verificationTimeseriesDataPoint>;
export type VerificationTimeseriesParams = z.infer<typeof verificationTimeseriesParams>;

type TimeInterval = {
  table: string;
  step: string;
  stepSize: number;
};

const INTERVALS: Record<string, TimeInterval> = {
  // Minute-based intervals
  minute: {
    table: "default.key_verifications_per_minute_v2",
    step: "MINUTE",
    stepSize: 1,
  },
  fiveMinutes: {
    table: "default.key_verifications_per_minute_v2",
    step: "MINUTE",
    stepSize: 5,
  },
  thirtyMinutes: {
    table: "default.key_verifications_per_minute_v2",
    step: "MINUTE",
    stepSize: 30,
  },
  // Hour-based intervals
  hour: {
    table: "default.key_verifications_per_hour_v2",
    step: "HOUR",
    stepSize: 1,
  },
  twoHours: {
    table: "default.key_verifications_per_hour_v2",
    step: "HOUR",
    stepSize: 2,
  },
  fourHours: {
    table: "default.key_verifications_per_hour_v2",
    step: "HOUR",
    stepSize: 4,
  },
  sixHours: {
    table: "default.key_verifications_per_hour_v2",
    step: "HOUR",
    stepSize: 6,
  },
  twelveHours: {
    table: "default.key_verifications_per_hour_v2",
    step: "HOUR",
    stepSize: 12,
  },
  // Day-based intervals
  day: {
    table: "default.key_verifications_per_day_v2",
    step: "DAY",
    stepSize: 1,
  },
  threeDays: {
    table: "default.key_verifications_per_day_v2",
    step: "DAY",
    stepSize: 3,
  },
  week: {
    table: "default.key_verifications_per_day_v2",
    step: "DAY",
    stepSize: 7,
  },
  twoWeeks: {
    table: "default.key_verifications_per_day_v2",
    step: "DAY",
    stepSize: 14,
  },
  // Monthly-based intervals
  month: {
    table: "default.key_verifications_per_month_v2",
    step: "MONTH",
    stepSize: 1,
  },
  quarter: {
    table: "default.key_verifications_per_month_v2",
    step: "MONTH",
    stepSize: 3,
  },
} as const;

function createVerificationTimeseriesQuery(interval: TimeInterval, whereClause: string) {
  const intervalUnit = {
    MINUTE: "minute",
    HOUR: "hour",
    DAY: "day",
    MONTH: "month",
  }[interval.step];

  // For millisecond step calculation
  const msPerUnit = {
    MINUTE: 60_000,
    HOUR: 3600_000,
    DAY: 86400_000,
    MONTH: 2592000_000,
  }[interval.step];

  if (!msPerUnit) {
    throw new Error(
      `Unsupported interval step: ${interval.step}. Expected one of: MINUTE, HOUR, DAY, MONTH`,
    );
  }

  const stepMs = msPerUnit * interval.stepSize;

  return `
    SELECT
      toUnixTimestamp64Milli(CAST(toStartOfInterval(time, INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3))) as x,
    map(
      'total', SUM(count),
      'valid', SUM(IF(outcome = 'VALID', count, 0)),
      'rate_limited_count', SUM(IF(outcome = 'RATE_LIMITED', count, 0)),
      'insufficient_permissions_count', SUM(IF(outcome = 'INSUFFICIENT_PERMISSIONS', count, 0)),
      'forbidden_count', SUM(IF(outcome = 'FORBIDDEN', count, 0)),
      'disabled_count', SUM(IF(outcome = 'DISABLED', count, 0)) ,
      'expired_count',SUM(IF(outcome = 'EXPIRED', count, 0)) ,
      'usage_exceeded_count', SUM(IF(outcome = 'USAGE_EXCEEDED', count, 0))
      ) as y
    FROM ${interval.table}
    ${whereClause}
    GROUP BY x
    ORDER BY x ASC
    WITH FILL
      FROM toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({startTime: Int64})), INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3)))
      TO toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({endTime: Int64})), INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3))) + ${stepMs}
      STEP ${stepMs}`;
}

function getVerificationTimeseriesWhereClause(
  params: VerificationTimeseriesParams,
  additionalConditions: string[] = [],
): { whereClause: string; paramSchema: z.ZodType<unknown> } {
  const conditions = [
    "workspace_id = {workspaceId: String}",
    "key_space_id = {keyspaceId: String}",
    ...additionalConditions,
  ];

  // Create parameter schema extension
  const paramSchemaExtension: Record<string, z.ZodType> = {};

  // Add keyId direct filter if specified
  if (params.keyId) {
    conditions.push("key_id = {keyId: String}");
  }

  // Add keyIds filter conditions
  if (params.keyIds?.length) {
    const keyIdConditions = params.keyIds
      .map((filter, index) => {
        const paramName = `keyIdValue_${index}`;
        paramSchemaExtension[paramName] = z.string();

        switch (filter.operator) {
          case "is":
            return `key_id = {${paramName}: String}`;
          case "contains":
            return `like(key_id, CONCAT('%', {${paramName}: String}, '%'))`;
        }
      })
      .filter(Boolean)
      .join(" OR ");

    if (keyIdConditions.length > 0) {
      conditions.push(`(${keyIdConditions})`);
    }
  }

  // Add outcomes filter conditions
  if (params.outcomes?.length) {
    const outcomeConditions = params.outcomes
      .map((filter, index) => {
        const paramName = `outcomeValue_${index}`;
        paramSchemaExtension[paramName] = z.string();

        if (filter.operator === "is") {
          return `outcome = {${paramName}: String}`;
        }
        return null;
      })
      .filter(Boolean)
      .join(" OR ");

    if (outcomeConditions.length > 0) {
      conditions.push(`(${outcomeConditions})`);
    }
  }

  // Handle tags filter
  if (params.tags && params.tags.length > 0) {
    const tagConditions = params.tags
      .map((filter, index) => {
        const paramName = `tagValue_${index}`;
        paramSchemaExtension[paramName] = z.string();

        switch (filter.operator) {
          case "is":
            return `has(tags, {${paramName}: String})`;
          case "contains":
            return `arrayExists(tag -> position(tag, {${paramName}: String}) > 0, tags)`;
          case "startsWith":
            return `arrayExists(tag -> startsWith(tag, {${paramName}: String}), tags)`;
          case "endsWith":
            return `arrayExists(tag -> endsWith(tag, {${paramName}: String}), tags)`;
          default:
            return null;
        }
      })
      .filter(Boolean);

    if (tagConditions.length > 0) {
      conditions.push(`(${tagConditions.join(" AND ")})`);
    }
  }

  return {
    whereClause: conditions.length > 0 ? `WHERE ${conditions.join(" AND ")}` : "",
    paramSchema: verificationTimeseriesParams.extend(paramSchemaExtension),
  };
}

function createVerificationTimeseriesQuerier(interval: TimeInterval) {
  return (ch: Querier) => async (args: VerificationTimeseriesParams) => {
    const { whereClause, paramSchema } = getVerificationTimeseriesWhereClause(args, [
      "time >= fromUnixTimestamp64Milli({startTime: Int64})",
      "time <= fromUnixTimestamp64Milli({endTime: Int64})",
    ]);

    // Create parameters object with filter values
    const parameters = {
      ...args,
      ...(args.keyIds?.reduce(
        (acc, filter, index) => ({
          // biome-ignore lint/performance/noAccumulatingSpread: We don't care about the spread syntax warning here
          ...acc,
          [`keyIdValue_${index}`]: filter.value,
        }),
        {},
      ) ?? {}),
      ...(args.tags?.reduce(
        (acc, filter, index) => ({
          // biome-ignore lint/performance/noAccumulatingSpread: We don't care about the spread syntax warning here
          ...acc,
          [`tagValue_${index}`]: filter.value,
        }),
        {},
      ) ?? {}),
      ...(args.outcomes?.reduce(
        (acc, filter, index) => ({
          // biome-ignore lint/performance/noAccumulatingSpread: We don't care about the spread syntax warning here
          ...acc,
          [`outcomeValue_${index}`]: filter.value,
        }),
        {},
      ) ?? {}),
    };

    return ch.query({
      query: createVerificationTimeseriesQuery(interval, whereClause),
      params: paramSchema,
      schema: verificationTimeseriesDataPoint,
    })(parameters);
  };
}

async function batchVerificationTimeseries(
  ch: Querier,
  interval: TimeInterval,
  args: VerificationTimeseriesParams,
  maxBatchSize = 15,
) {
  if (!args.keyIds || args.keyIds.length === 0 || args.keyIds.length <= maxBatchSize) {
    return (await createVerificationTimeseriesQuerier(interval)(ch)(args)).val;
  }

  const keyIdBatches: { value: string; operator: "is" | "contains" }[][] = [];
  for (let i = 0; i < args.keyIds.length; i += maxBatchSize) {
    keyIdBatches.push(args.keyIds.slice(i, i + maxBatchSize));
  }

  const batchResults = await Promise.allSettled(
    keyIdBatches.map(async (batchKeyIds, batchIndex) => {
      const batchArgs = {
        ...args,
        keyIds: batchKeyIds,
      };
      try {
        const res = await createVerificationTimeseriesQuerier(interval)(ch)(batchArgs);
        if (res?.val) {
          return res.val;
        }
        return res; // Return res directly if no .val
      } catch (error) {
        console.error(`Batch ${batchIndex} query failed:`, error);
        return [];
      }
    }),
  );

  const successfulResults = batchResults
    .filter((result) => result.status === "fulfilled")
    .map((result) => (result as PromiseFulfilledResult<VerificationTimeseriesDataPoint[]>).value)
    .filter((value) => Array.isArray(value));

  return mergeVerificationTimeseriesResults(successfulResults);
}

function mergeVerificationTimeseriesResults(
  results: VerificationTimeseriesDataPoint[][],
): VerificationTimeseriesDataPoint[] {
  const mergedMap = new Map<number, VerificationTimeseriesDataPoint>();

  results.forEach((resultBatch) => {
    resultBatch.forEach((dataPoint) => {
      if (!dataPoint) {
        return; // Skip undefined or null points
      }
      const existingPoint = mergedMap.get(dataPoint.x);

      if (existingPoint) {
        mergedMap.set(dataPoint.x, {
          x: dataPoint.x,
          y: {
            total: (existingPoint.y.total ?? 0) + (dataPoint.y.total ?? 0),
            valid: (existingPoint.y.valid ?? 0) + (dataPoint.y.valid ?? 0),
            valid_count: (existingPoint.y.valid_count ?? 0) + (dataPoint.y.valid_count ?? 0),
            rate_limited_count:
              (existingPoint.y.rate_limited_count ?? 0) + (dataPoint.y.rate_limited_count ?? 0),
            insufficient_permissions_count:
              (existingPoint.y.insufficient_permissions_count ?? 0) +
              (dataPoint.y.insufficient_permissions_count ?? 0),
            forbidden_count:
              (existingPoint.y.forbidden_count ?? 0) + (dataPoint.y.forbidden_count ?? 0),
            disabled_count:
              (existingPoint.y.disabled_count ?? 0) + (dataPoint.y.disabled_count ?? 0),
            expired_count: (existingPoint.y.expired_count ?? 0) + (dataPoint.y.expired_count ?? 0),
            usage_exceeded_count:
              (existingPoint.y.usage_exceeded_count ?? 0) + (dataPoint.y.usage_exceeded_count ?? 0),
          },
        });
      } else {
        mergedMap.set(dataPoint.x, dataPoint);
      }
    });
  });

  // Convert map back to sorted array
  return Array.from(mergedMap.values()).sort((a, b) => a.x - b.x);
}

// Minute-based timeseries
export const getMinutelyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.minute, args);
export const getFiveMinutelyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.fiveMinutes, args);
export const getThirtyMinutelyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.thirtyMinutes, args);

// Hour-based timeseries
export const getHourlyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.hour, args);
export const getTwoHourlyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.twoHours, args);
export const getFourHourlyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.fourHours, args);
export const getSixHourlyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.sixHours, args);
export const getTwelveHourlyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.twelveHours, args);

// Day-based timeseries
export const getDailyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.day, args);
export const getThreeDayVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.threeDays, args);
export const getWeeklyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.week, args);

// Month-based timeseries
export const getMonthlyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.month, args);
