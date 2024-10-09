import { z } from "zod";
import { clickhouse } from "./client";

// clickhouse DateTime returns a string, which we need to parse
const dateTimeToUnix = z.string().transform((t) => new Date(t).getTime());
const outcome = z.enum([
  "VALID",
  "INSUFFICIENT_PERMISSIONS",
  "RATE_LIMITED",
  "FORBIDDEN",
  "DISABLED",
  "EXPIRED",
  "USAGE_EXCEEDED",
]);

const params = z.object({
  workspaceId: z.string(),
  keySpaceId: z.string(),
  start: z.number().int(),
  end: z.number().int(),
});
const schema = z.object({
  time: dateTimeToUnix,
  outcome,
  count: z.number().int(),
});

export async function getVerificationsPerHour(args: {
  workspaceId: string;
  keySpaceId: string;
  keyId?: string;
  start: number;
  end: number;
}) {
  const query = `
    SELECT 
      time,
      outcome,
      count
    FROM verifications.key_verifications_per_hour_v1
    WHERE 
      workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String} AND time >= {start: Int64}
    AND time < {end: Int64}
    ${args.keyId ? "AND key_id = {keyId: String}" : ""}
    GROUP BY time
    ORDER BY time ASC
    WITH FILL 
      FROM toStartOfHour(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfHour(fromUnixTimestamp64Milli({end: Int65}))
      STEP INTERVAL 1 HOUR
    ;`;

  return clickhouse.query({
    query,
    params,
    schema,
  })(args);
}

export async function getVerificationsPerDay(args: {
  workspaceId: string;
  keySpaceId: string;
  keyId?: string;
  start: number;
  end: number;
}) {
  const query = `
    SELECT 
      time,
      outcome,
      count
    FROM verifications.key_verifications_per_day_v1
    WHERE 
      workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String} AND time >= {start: Int64}
    AND time < {end: Int64}
    ${args.keyId ? "AND key_id = {keyId: String}" : ""}
    GROUP BY time
    ORDER BY time ASC
    WITH FILL 
      FROM toStartOfDay(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfDay(fromUnixTimestamp64Milli({end: Int65}))
      STEP INTERVAL 1DAY
    ;`;

  return clickhouse.query({ query, params, schema })(args);
}

export async function getVerificationsPerMonth(args: {
  workspaceId: string;
  keySpaceId: string;
  keyId?: string;
  start: number;
  end: number;
}) {
  const query = `
    SELECT 
      time,
      outcome,
      count
    FROM verifications.key_verifications_per_month_v1
    WHERE 
      workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String} AND time >= {start: Int64}
    AND time < {end: Int64}
    ${args.keyId ? "AND key_id = {keyId: String}" : ""}
    GROUP BY time
    ORDER BY time ASC
    WITH FILL 
      FROM toStartOfMonth(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfMonth(fromUnixTimestamp64Milli({end: Int65}))
      STEP INTERVAL 1 MONTH
    ;`;

  return clickhouse.query({
    query,
    params,
    schema,
  })(args);
}
