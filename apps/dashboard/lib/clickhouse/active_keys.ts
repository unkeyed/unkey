import { type Clickhouse, Client, Noop } from "@unkey/clickhouse-zod";
import { z } from "zod";
import { env } from "../env";

export async function getActiveKeysPerHour(args: {
  workspaceId: string;
  keySpaceId: string;
  start: number;
  end: number;
}) {
  const { CLICKHOUSE_URL } = env();

  const ch: Clickhouse = CLICKHOUSE_URL ? new Client({ url: CLICKHOUSE_URL }) : new Noop();
  const query = ch.query({
    query: `
    SELECT
      count(DISTINCT keyId) as keys,
      time,
    FROM verifications.key_verifications_per_hour_v1
    WHERE 
      workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String} AND time >= {start: Int64}
    AND time < {end: Int64}
    GROUP BY time
    ORDER BY time ASC
    WITH FILL 
      FROM toStartOfHour(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfHour(fromUnixTimestamp64Milli({end: Int65}))
      STEP INTERVAL 1 HOUR
    ;`,
    params: z.object({
      workspaceId: z.string(),
      keySpaceId: z.string(),
      start: z.number().int(),
      end: z.number().int(),
    }),
    schema: z.object({
      keys: z.number().int(),
      time: z.number().int(),
    }),
  });

  return query(args);
}

export async function getActiveKeysPerDay(args: {
  workspaceId: string;
  keySpaceId: string;
  start: number;
  end: number;
}) {
  const { CLICKHOUSE_URL } = env();

  const ch: Clickhouse = CLICKHOUSE_URL ? new Client({ url: CLICKHOUSE_URL }) : new Noop();
  const query = ch.query({
    query: `
    SELECT
      count(DISTINCT keyId) as keys,
      time,
    FROM verifications.key_verifications_per_day_v1
    WHERE 
      workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String}
    AND time >= {start: Int64}
    AND time < {end: Int64}
    GROUP BY time
    ORDER BY time ASC
    WITH FILL 
      FROM toStartOfDay(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfDay(fromUnixTimestamp64Milli({end: Int65}))
      STEP INTERVAL 1 DAY
    ;`,
    params: z.object({
      workspaceId: z.string(),
      keySpaceId: z.string(),
      start: z.number().int(),
      end: z.number().int(),
    }),
    schema: z.object({
      keys: z.number().int(),
      time: z.number().int(),
    }),
  });

  return query(args);
}
export async function getActiveKeysPerMonth(args: {
  workspaceId: string;
  keySpaceId: string;
  start: number;
  end: number;
}) {
  const { CLICKHOUSE_URL } = env();

  const ch: Clickhouse = CLICKHOUSE_URL ? new Client({ url: CLICKHOUSE_URL }) : new Noop();
  const query = ch.query({
    query: `
    SELECT
      count(DISTINCT keyId) as keys,
      time,
    FROM verifications.key_verifications_per_month_v1
    WHERE 
      workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String}
    AND time >= {start: Int64}
    AND time < {end: Int64}
    GROUP BY time
    ORDER BY time ASC
    WITH FILL 
      FROM toStartOfMonth(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfMonth(fromUnixTimestamp64Milli({end: Int65}))
      STEP INTERVAL 1 MONTH
    ;`,
    params: z.object({
      workspaceId: z.string(),
      keySpaceId: z.string(),
      start: z.number().int(),
      end: z.number().int(),
    }),
    schema: z.object({
      keys: z.number().int(),
      time: z.number().int(),
    }),
  });

  return query(args);
}
