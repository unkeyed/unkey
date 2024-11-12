import { z } from "zod";
import type { Inserter, Querier } from "./client";
import { dateTimeToUnix } from "./util";

const params = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  identifier: z.array(z.string()).optional(),
  start: z.number().default(0),
  end: z.number().default(() => Date.now()),
});

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

export function getRatelimitsPerMinute(ch: Querier) {
  return async (args: z.input<typeof params>) => {
    const query = ch.query({
      query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_minute_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}) > 0" : ""}
      AND time >= fromUnixTimestamp64Milli({start: Int64})
      AND time <= fromUnixTimestamp64Milli({end: Int64})
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfMinute(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfMinute(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 MINUTE
;`,
      params,
      schema: z.object({
        time: dateTimeToUnix,
        passed: z.number(),
        total: z.number(),
      }),
    });

    return query(args);
  };
}

export function getRatelimitsPerHour(ch: Querier) {
  return async (args: z.infer<typeof params>) => {
    const query = ch.query({
      query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_hour_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}) > 0" : ""}
      AND time >= fromUnixTimestamp64Milli({start: Int64})
      AND time <= fromUnixTimestamp64Milli({end: Int64})
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfHour(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfHour(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 HOUR
;`,
      params,
      schema: z.object({
        time: dateTimeToUnix,
        passed: z.number(),
        total: z.number(),
      }),
    });

    return query(args);
  };
}
export function getRatelimitsPerDay(ch: Querier) {
  return async (args: z.infer<typeof params>) => {
    const query = ch.query({
      query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_day_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}) > 0" : ""}
      AND time >= fromUnixTimestamp64Milli({start: Int64})
      AND time <= fromUnixTimestamp64Milli({end: Int64})
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfDay(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfDay(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 DAY
;`,
      params,
      schema: z.object({
        time: dateTimeToUnix,
        passed: z.number(),
        total: z.number(),
      }),
    });

    return query(args);
  };
}
export function getRatelimitsPerMonth(ch: Querier) {
  return async (args: z.input<typeof params>) => {
    const query = ch.query({
      query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_month_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}) > 0" : ""}
      AND time >= fromUnixTimestamp64Milli({start: Int64})
      AND time <= fromUnixTimestamp64Milli({end: Int64})
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfMonth(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfMonth(fromUnixTimestamp64Milli({end: Int64}))
      STEP INTERVAL 1 MONTH
;`,
      params,
      schema: z.object({
        time: dateTimeToUnix,
        passed: z.number(),
        total: z.number(),
      }),
    });

    return query(args);
  };
}
const getRatelimitLogsParameters = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  identifier: z.array(z.string()).optional(),
  start: z.number().optional().default(0),
  end: z
    .number()
    .optional()
    .default(() => Date.now()),
  limit: z.number().optional().default(100),
});

export function getRatelimitLogs(ch: Querier) {
  return async (args: z.input<typeof getRatelimitLogsParameters>) => {
    const query = ch.query({
      query: `
    SELECT
      request_id,
      time,
      identifier,
      passed
    FROM ratelimits.raw_ratelimits
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}) > 0" : ""}
      AND time >= {start: Int64}
      AND time <= {end: Int64}
    LIMIT {limit: Int64}
;`,
      params: getRatelimitLogsParameters,
      schema: z.object({
        request_id: z.string(),
        time: z.number(),
        identifier: z.string(),
        passed: z.boolean(),
      }),
    });

    return query(args);
  };
}
const getRatelimitLastUsedParameters = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  identifier: z.array(z.string()).optional(),
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
