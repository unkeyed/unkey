import { z } from "zod";
import type { Inserter, Querier } from "./client";
import { dateTimeToUnix } from "./util";

const outcome = z.enum([
  "VALID",
  "INSUFFICIENT_PERMISSIONS",
  "RATE_LIMITED",
  "FORBIDDEN",
  "DISABLED",
  "EXPIRED",
  "USAGE_EXCEEDED",
  "",
]);

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
});

export const verificationTimeseriesDataPoint = z.object({
  x: z.number().int(),
  y: z.object({
    total: z.number().int().default(0),
    valid: z.number().int().default(0),
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

  // Calculate step in milliseconds
  const stepMs = msPerUnit! * interval.stepSize;

  return `
    SELECT
      toUnixTimestamp64Milli(CAST(toStartOfInterval(time, INTERVAL ${interval.stepSize} ${intervalUnit}) AS DateTime64(3))) as x,
      map(
          'total', SUM(count),
          'valid', SUM(IF(outcome = 'VALID', count, 0))
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

function getVerificationTimeseriesWhereClause(): string {
  const conditions = [
    "workspace_id = {workspaceId: String}",
    "key_space_id = {keyspaceId: String}",
    "time >= fromUnixTimestamp64Milli({startTime: Int64})",
    "time <= fromUnixTimestamp64Milli({endTime: Int64})",
  ];

  return conditions.length > 0 ? `WHERE ${conditions.join(" AND ")}` : "";
}

function createVerificationTimeseriesQuerier(interval: TimeInterval) {
  return (ch: Querier) => async (args: VerificationTimeseriesParams) => {
    const whereClause = getVerificationTimeseriesWhereClause();
    const query = createVerificationTimeseriesQuery(interval, whereClause);

    return ch.query({
      query,
      params: verificationTimeseriesParams,
      schema: verificationTimeseriesDataPoint,
    })(args);
  };
}

export const getHourlyVerificationTimeseries = createVerificationTimeseriesQuerier(INTERVALS.hour);
export const getTwoHourlyVerificationTimeseries = createVerificationTimeseriesQuerier(
  INTERVALS.twoHours,
);
export const getFourHourlyVerificationTimeseries = createVerificationTimeseriesQuerier(
  INTERVALS.fourHours,
);
export const getSixHourlyVerificationTimeseries = createVerificationTimeseriesQuerier(
  INTERVALS.sixHours,
);
export const getTwelveHourlyVerificationTimeseries = createVerificationTimeseriesQuerier(
  INTERVALS.twelveHours,
);
export const getDailyVerificationTimeseries = createVerificationTimeseriesQuerier(INTERVALS.day);
export const getThreeDayVerificationTimeseries = createVerificationTimeseriesQuerier(
  INTERVALS.threeDays,
);
export const getWeeklyVerificationTimeseries = createVerificationTimeseriesQuerier(INTERVALS.week);
export const getMonthlyVerificationTimeseries = createVerificationTimeseriesQuerier(
  INTERVALS.month,
);
