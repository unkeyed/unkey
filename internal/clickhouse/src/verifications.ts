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
