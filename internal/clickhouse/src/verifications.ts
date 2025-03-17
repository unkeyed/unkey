import { z } from "zod";
import type { Inserter, Querier } from "./client";
import { KEY_VERIFICATION_OUTCOMES } from "./keys/keys";
import { dateTimeToUnix } from "./util";

const outcome = z.enum(KEY_VERIFICATION_OUTCOMES);

const params = z.object({
  workspaceId: z.string(),
  keySpaceId: z.string(),
  keyId: z.string().optional(),
  start: z.number().int(),
  end: z.number().int(),
});
const schema = z.object({
  time: dateTimeToUnix,
  outcome,
  count: z.number().int(),
  tags: z.array(z.string()),
});

export function insertVerification(ch: Inserter) {
  return ch.insert({
    table: "verifications.raw_key_verifications_v1",
    schema: z.object({
      request_id: z.string(),
      time: z.number().int(),
      workspace_id: z.string(),
      key_space_id: z.string(),
      key_id: z.string(),
      region: z.string(),
      tags: z.array(z.string()).transform((arr) => arr.sort()),
      outcome: z.enum([
        "VALID",
        "RATE_LIMITED",
        "EXPIRED",
        "DISABLED",
        "FORBIDDEN",
        "USAGE_EXCEEDED",
        "INSUFFICIENT_PERMISSIONS",
      ]),
      identity_id: z.string().optional().default(""),
    }),
  });
}

export function getVerificationsPerHour(ch: Querier) {
  return async (args: z.input<typeof params>) => {
    const query = `
    SELECT
      time,
      outcome,
      sum(count) as count,
      tags
    FROM verifications.key_verifications_per_hour_v3
    WHERE
      workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String}
    AND time >= fromUnixTimestamp64Milli({start: Int64})
    AND time <= fromUnixTimestamp64Milli({end: Int64})
    ${args.keyId ? "AND key_id = {keyId: String}" : ""}
    GROUP BY time, outcome, tags
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfHour(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfHour(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 HOUR
    ;`;

    return ch.query({
      query,
      params,
      schema,
    })(args);
  };
}

export function getVerificationsPerDay(ch: Querier) {
  return async (args: z.input<typeof params>) => {
    const query = `
    SELECT
      time,
      outcome,
      sum(count) as count,
      tags
    FROM verifications.key_verifications_per_day_v3
    WHERE
      workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String}
    AND time >= fromUnixTimestamp64Milli({start: Int64})
    AND time <= fromUnixTimestamp64Milli({end: Int64})
    ${args.keyId ? "AND key_id = {keyId: String}" : ""}
    GROUP BY time, outcome, tags
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfDay(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfDay(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 DAY
    ;`;

    return ch.query({ query, params, schema })(args);
  };
}

export function getVerificationsPerMonth(ch: Querier) {
  return async (args: z.input<typeof params>) => {
    const query = `
    SELECT
      time,
      outcome,
      sum(count) as count,
      tags
    FROM verifications.key_verifications_per_month_v3
    WHERE
      workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String}
    AND time >= fromUnixTimestamp64Milli({start: Int64})
    AND time <= fromUnixTimestamp64Milli({end: Int64})
    ${args.keyId ? "AND key_id = {keyId: String}" : ""}
    GROUP BY time, outcome, tags
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfDay(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfDay(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 MONTH
    ;`;

    return ch.query({ query, params, schema })(args);
  };
}

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
  hour: {
    table: "verifications.key_verifications_per_hour_v3",
    step: "HOUR",
    stepSize: 1,
  },
  twoHours: {
    table: "verifications.key_verifications_per_hour_v3",
    step: "HOURS",
    stepSize: 2,
  },
  fourHours: {
    table: "verifications.key_verifications_per_hour_v3",
    step: "HOURS",
    stepSize: 4,
  },
  sixHours: {
    table: "verifications.key_verifications_per_hour_v3",
    step: "HOURS",
    stepSize: 6,
  },
  twelveHours: {
    table: "verifications.key_verifications_per_hour_v3",
    step: "HOURS",
    stepSize: 12,
  },
  day: {
    table: "verifications.key_verifications_per_day_v3",
    step: "DAY",
    stepSize: 1,
  },
  threeDays: {
    table: "verifications.key_verifications_per_day_v3",
    step: "DAYS",
    stepSize: 3,
  },
  week: {
    table: "verifications.key_verifications_per_day_v3",
    step: "DAYS",
    stepSize: 7,
  },
  twoWeeks: {
    table: "verifications.key_verifications_per_day_v3",
    step: "DAYS",
    stepSize: 14,
  },
  // Monthly-based intervals
  month: {
    table: "verifications.key_verifications_per_month_v3",
    step: "MONTH",
    stepSize: 1,
  },
  quarter: {
    table: "verifications.key_verifications_per_month_v3",
    step: "MONTHS",
    stepSize: 3,
  },
} as const;

function createVerificationTimeseriesQuery(interval: TimeInterval, whereClause: string) {
  const intervalUnit = {
    HOUR: "hour",
    HOURS: "hour",
    DAY: "day",
    DAYS: "day",
    MONTH: "month",
    MONTHS: "month",
  }[interval.step];

  // For millisecond step calculation
  const msPerUnit = {
    HOUR: 3600_000,
    HOURS: 3600_000,
    DAY: 86400_000,
    DAYS: 86400_000,
    MONTH: 2592000_000,
    MONTHS: 2592000_000,
  }[interval.step];

  const stepMs = msPerUnit! * interval.stepSize;

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
      TO toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({endTime: Int64})), INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3)))
      STEP ${stepMs}`;
}

function getVerificationTimeseriesWhereClause(
  params: VerificationTimeseriesParams,
  additionalConditions: string[] = [],
): { whereClause: string; paramSchema: z.ZodType<any> } {
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

  return {
    whereClause: conditions.length > 0 ? `WHERE ${conditions.join(" AND ")}` : "",
    paramSchema: verificationTimeseriesParams.extend(paramSchemaExtension),
  };
}

// Updated timeseries querier function
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
          // biome-ignore lint/performance/noAccumulatingSpread: <explanation>
          ...acc,
          [`keyIdValue_${index}`]: filter.value,
        }),
        {},
      ) ?? {}),
      ...(args.outcomes?.reduce(
        (acc, filter, index) => ({
          // biome-ignore lint/performance/noAccumulatingSpread: <explanation>
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

  const keyIdBatches: any[] = [];
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

      if (!existingPoint) {
        mergedMap.set(dataPoint.x, dataPoint);
      } else {
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
      }
    });
  });

  // Convert map back to sorted array
  return Array.from(mergedMap.values()).sort((a, b) => a.x - b.x);
}

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

export const getDailyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.day, args);

export const getThreeDayVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.threeDays, args);

export const getWeeklyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.week, args);

export const getMonthlyVerificationTimeseries =
  (ch: Querier) => (args: VerificationTimeseriesParams) =>
    batchVerificationTimeseries(ch, INTERVALS.month, args);
