import { z } from "zod";
import type { Querier } from "./client/interface";

export function getActiveKeysPerHour(ch: Querier) {
  return async (args: {
    workspaceId: string;
    keySpaceId: string;
    start: number;
    end: number;
  }) => {
    const query = ch.query({
      query: ` SELECT count(DISTINCT keyId) as keys, time, FROM verifications.key_verifications_per_hour_v1 WHERE workspace_id = {workspaceId: String} AND key_space_id = {keySpaceId: String} AND time >= {start: Int64} AND time < {end: Int64} GROUP BY time
    ORDER BY time ASC
    WITH FILL 
      FROM toStartOfHour(fromUnixTimestamp64Milli({start: Int64}))
      TO toStartOfHour(fromUnixTimestamp64Milli({end: Int64}))
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
  };
}

export function getActiveKeysPerDay(ch: Querier) {
  return async (args: {
    workspaceId: string;
    keySpaceId: string;
    start: number;
    end: number;
  }) => {
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
      TO toStartOfDay(fromUnixTimestamp64Milli({end: Int64}))
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
  };
}
export function getActiveKeysPerMonth(ch: Querier) {
  return async (args: {
    workspaceId: string;
    keySpaceId: string;
    start: number;
    end: number;
  }) => {
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
      TO toStartOfMonth(fromUnixTimestamp64Milli({end: Int64}))
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
  };
}
