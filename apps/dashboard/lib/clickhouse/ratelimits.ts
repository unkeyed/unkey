import { type Clickhouse, Client, Noop } from "@unkey/clickhouse-zod";
import { z } from "zod";
import { env } from "../env";
import { dateTimeToUnix } from "./util";

const params = z.object({
  workspaceId: z.string(),
  namespaceId: z.string(),
  identifier: z.array(z.string()).optional(),
  start: z.number().default(0),
  end: z.number().default(() => Date.now()),
});

export async function getRatelimitsPerMinute(args: z.infer<typeof params>) {
  const { CLICKHOUSE_URL } = env();

  const ch: Clickhouse = CLICKHOUSE_URL ? new Client({ url: CLICKHOUSE_URL }) : new Noop();
  const query = ch.query({
    query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_minute_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}" : ""}
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
}

export async function getRatelimitsPerHour(args: z.infer<typeof params>) {
  const { CLICKHOUSE_URL } = env();

  const ch: Clickhouse = CLICKHOUSE_URL ? new Client({ url: CLICKHOUSE_URL }) : new Noop();
  const query = ch.query({
    query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_hour_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}" : ""}
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
}

export async function getRatelimitsPerDay(args: z.infer<typeof params>) {
  const { CLICKHOUSE_URL } = env();

  const ch: Clickhouse = CLICKHOUSE_URL ? new Client({ url: CLICKHOUSE_URL }) : new Noop();
  const query = ch.query({
    query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_day_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}" : ""}
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
}
export async function getRatelimitsPerMonth(args: z.infer<typeof params>) {
  const { CLICKHOUSE_URL } = env();

  const ch: Clickhouse = CLICKHOUSE_URL ? new Client({ url: CLICKHOUSE_URL }) : new Noop();
  const query = ch.query({
    query: `
    SELECT
      time,
      sum(passed) as passed,
      sum(total) as total
    FROM ratelimits.ratelimits_per_month_v1
    WHERE workspace_id = {workspaceId: String}
      AND namespace_id = {namespaceId: String}
      ${args.identifier ? "AND multiSearchAny(identifier, {identifier: Array(String)}" : ""}
      AND time >= fromUnixTimestamp64Milli({start: Int64})
      AND time <= fromUnixTimestamp64Milli({end: Int64})
    GROUP BY time
    ORDER BY time ASC
    WITH FILL
      FROM toStartOfMonth(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfMonth(fromUnixTimestamp64Milli({end: Int64})
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
}
