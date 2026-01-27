import { z } from "zod";
import type { Querier } from "../client";
import { KEY_VERIFICATION_OUTCOMES } from "./keys";

export const activeKeysTimeseriesParams = z.object({
  workspaceId: z.string(),
  keyspaceId: z.string(),
  keyId: z.string().optional(),
  startTime: z.int(),
  endTime: z.int(),
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

export const activeKeysTimeseriesDataPoint = z.object({
  x: z.int(),
  y: z.object({
    keys: z.int().prefault(0),
  }),
  key_ids: z.array(z.string()).optional(),
});

export type ActiveKeysTimeseriesDataPoint = z.infer<typeof activeKeysTimeseriesDataPoint>;
export type ActiveKeysTimeseriesParams = z.infer<typeof activeKeysTimeseriesParams>;

type TimeInterval = {
  table: string;
  step: string;
  stepSize: number;
};

const ACTIVE_KEYS_INTERVALS: Record<string, TimeInterval> = {
  // Minute-based intervals
  minute: {
    table: "default.key_verifications_per_minute_v3",
    step: "MINUTE",
    stepSize: 1,
  },
  fiveMinutes: {
    table: "default.key_verifications_per_minute_v3",
    step: "MINUTE",
    stepSize: 5,
  },
  fifteenMinutes: {
    table: "default.key_verifications_per_minute_v3",
    step: "MINUTE",
    stepSize: 15,
  },
  thirtyMinutes: {
    table: "default.key_verifications_per_minute_v3",
    step: "MINUTE",
    stepSize: 30,
  },
  // Hour-based intervals
  hour: {
    table: "default.key_verifications_per_hour_v3",
    step: "HOUR",
    stepSize: 1,
  },
  twoHours: {
    table: "default.key_verifications_per_hour_v3",
    step: "HOUR",
    stepSize: 2,
  },
  fourHours: {
    table: "default.key_verifications_per_hour_v3",
    step: "HOUR",
    stepSize: 4,
  },
  sixHours: {
    table: "default.key_verifications_per_hour_v3",
    step: "HOUR",
    stepSize: 6,
  },
  twelveHours: {
    table: "default.key_verifications_per_hour_v3",
    step: "HOUR",
    stepSize: 12,
  },
  // Day-based intervals
  day: {
    table: "default.key_verifications_per_day_v3",
    step: "DAY",
    stepSize: 1,
  },
  threeDays: {
    table: "default.key_verifications_per_day_v3",
    step: "DAY",
    stepSize: 3,
  },
  week: {
    table: "default.key_verifications_per_day_v3",
    step: "DAY",
    stepSize: 7,
  },
  twoWeeks: {
    table: "default.key_verifications_per_day_v3",
    step: "DAY",
    stepSize: 14,
  },
  // Monthly-based intervals
  month: {
    table: "default.key_verifications_per_month_v3",
    step: "MONTH",
    stepSize: 1,
  },
  quarter: {
    table: "default.key_verifications_per_month_v3",
    step: "MONTH",
    stepSize: 3,
  },
} as const;

function createActiveKeysTimeseriesQuery(interval: TimeInterval, whereClause: string) {
  const intervalUnit = {
    MINUTE: "minute",
    HOUR: "hour",
    DAY: "day",
    DAYS: "day",
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
    throw new Error("Unknown interval in 'createActiveKeysTimeseriesQuery'");
  }

  const stepMs = msPerUnit * interval.stepSize;

  return `
    SELECT
      toUnixTimestamp64Milli(CAST(toStartOfInterval(time, INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3))) as x,
      map(
        'keys', count(DISTINCT key_id)
      ) as y,
      groupArray(DISTINCT key_id) as key_ids
    FROM ${interval.table}
    ${whereClause}
    GROUP BY x
    ORDER BY x ASC
    WITH FILL
      FROM toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({startTime: Int64})), INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3)))
      TO toUnixTimestamp64Milli(CAST(toStartOfInterval(toDateTime(fromUnixTimestamp64Milli({endTime: Int64})), INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3))) + ${stepMs}
      STEP ${stepMs}`;
}

function getActiveKeysTimeseriesWhereClause(
  params: ActiveKeysTimeseriesParams,
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

  return {
    whereClause: conditions.length > 0 ? `WHERE ${conditions.join(" AND ")}` : "",
    paramSchema: activeKeysTimeseriesParams.extend(paramSchemaExtension),
  };
}

// Create timeseries querier function for active keys
function createActiveKeysTimeseriesQuerier(interval: TimeInterval) {
  return (ch: Querier) => async (args: ActiveKeysTimeseriesParams) => {
    const { whereClause, paramSchema } = getActiveKeysTimeseriesWhereClause(args, [
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
      ...(args.names?.reduce(
        (acc, filter, index) => ({
          // biome-ignore lint/performance/noAccumulatingSpread: <explanation>
          ...acc,
          [`nameValue_${index}`]: filter.value,
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
      query: createActiveKeysTimeseriesQuery(interval, whereClause),
      params: paramSchema,
      schema: activeKeysTimeseriesDataPoint,
    })(parameters);
  };
}
// Minute-based timeseries
export const getMinutelyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.minute,
);
export const getFiveMinutelyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.fiveMinutes,
);
export const getFifteenMinutelyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.fifteenMinutes,
);
export const getThirtyMinutelyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.thirtyMinutes,
);

// Hour-based timeseries
export const getHourlyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.hour,
);
export const getTwoHourlyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.twoHours,
);
export const getFourHourlyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.fourHours,
);
export const getSixHourlyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.sixHours,
);
export const getTwelveHourlyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.twelveHours,
);

// Day-based timeseries
export const getDailyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.day,
);
export const getThreeDayActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.threeDays,
);
export const getWeeklyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.week,
);
export const getTwoWeeklyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.twoWeeks,
);

// Month-based timeseries
export const getMonthlyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.month,
);
export const getQuarterlyActiveKeysTimeseries = createActiveKeysTimeseriesQuerier(
  ACTIVE_KEYS_INTERVALS.quarter,
);
