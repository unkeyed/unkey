import { env } from "@/lib/env";
import { NoopTinybird, Tinybird } from "@chronark/zod-bird";
import { z } from "zod";

const token = env().TINYBIRD_TOKEN;
const tb = token ? new Tinybird({ token }) : new NoopTinybird();

const datetimeToUnixMilli = z.string().transform((t) => new Date(t).getTime());

/**
 * `t` has the format `2021-01-01 00:00:00`
 *
 * If we transform it as is, we get `1609459200000` which is `2021-01-01 01:00:00` due to fun timezone stuff.
 * So we split the string at the space and take the date part, and then parse that.
 */
const dateToUnixMilli = z.string().transform((t) => new Date(t.split(" ").at(0) ?? t).getTime());

export const getDailyVerifications = tb.buildPipe({
  pipe: "endpoint__get_daily_verifications__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string().optional(),
    keyId: z.string().optional(),
  }),
  data: z.object({
    time: datetimeToUnixMilli,
    success: z.number(),
    rateLimited: z.number(),
    usageExceeded: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getActiveCountPerApiPerDay = tb.buildPipe({
  pipe: "endpoint_get_active_keys__v2",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string().optional(),
    start: z.number(),
    end: z.number(),
  }),
  data: z.object({
    active: z.number(),
  }),
});

export const getTotalVerificationsForWorkspace = tb.buildPipe({
  pipe: "endpoint_billing_get_verifications_usage__v1",
  parameters: z.object({
    workspaceId: z.string(),
    start: z.number(),
    end: z.number(),
  }),
  data: z.object({ usage: z.number() }),
  opts: {
    cache: "no-store",
  },
});

export const getTotalActiveKeys = tb.buildPipe({
  pipe: "endpoint_billing_get_active_keys_usage__v2",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string().optional(),
    start: z.number(),
    end: z.number(),
  }),
  data: z.object({ usage: z.number() }),
  opts: {
    cache: "no-store",
  },
});

export const getTotalVerifications = tb.buildPipe({
  pipe: "endpoint__all_verifications__v1",
  data: z.object({ verifications: z.number() }),
  opts: {
    cache: "no-store",
  },
});

export const getLatestVerifications = tb.buildPipe({
  pipe: "endpoint__get_latest_verifications__v2",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    keyId: z.string(),
  }),
  data: z.object({
    time: z.number(),
    requestedResource: z.string(),
    ratelimited: z.number().transform((n) => n > 0),
    usageExceeded: z.number().transform((n) => n > 0),
    region: z.string(),
    userAgent: z.string(),
    ipAddress: z.string(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getTotalVerificationsForKey = tb.buildPipe({
  pipe: "endpoint__get_total_usage_for_key__v1",
  parameters: z.object({
    keyId: z.string(),
  }),
  data: z.object({ totalUsage: z.number() }),
  opts: {
    cache: "no-store",
  },
});

export const getLastUsed = tb.buildPipe({
  pipe: "endpoint__get_last_used__v1",
  parameters: z.object({
    keyId: z.string(),
  }),
  data: z.object({
    lastUsed: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getActiveKeysPerHourForAllWorkspaces = tb.buildPipe({
  pipe: "endpoint_billing_get_active_keys_per_workspace_per_hour__v2__v1",

  data: z.object({
    usage: z.number(),
    workspaceId: z.string(),
    time: datetimeToUnixMilli,
  }),
  opts: {
    cache: "no-store",
  },
});

export const getVerificationsPerHourForAllWorkspaces = tb.buildPipe({
  pipe: "endpoint__billing_verifications_per_hour__v1",

  data: z.object({
    verifications: z.number(),
    workspaceId: z.string(),
    time: datetimeToUnixMilli,
  }),
  opts: {
    cache: "no-store",
  },
});

export const activeKeys = tb.buildPipe({
  pipe: "endpoint__active_keys_by_workspace__v1",
  parameters: z.object({
    workspaceId: z.string(),
    year: z.number().int(),
    month: z.number().int().min(1).max(12),
  }),
  data: z.object({
    keys: z.number().int().nullable().default(0),
  }),
  opts: {
    cache: "no-store",
  },
});

export const verifications = tb.buildPipe({
  pipe: "endpoint__verifications_by_workspace__v1",
  parameters: z.object({
    workspaceId: z.string(),
    year: z.number().int(),
    month: z.number().int().min(1).max(12),
  }),

  data: z.object({
    success: z.number().int().nullable().default(0),
    ratelimited: z.number().int().nullable().default(0),
    usageExceeded: z.number().int().nullable().default(0),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getVerificationsMonthly = tb.buildPipe({
  pipe: "get_verifications_monthly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    keyId: z.string().optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    success: z.number(),
    rateLimited: z.number(),
    usageExceeded: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getVerificationsWeekly = tb.buildPipe({
  pipe: "get_verifications_weekly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    keyId: z.string().optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    success: z.number(),
    rateLimited: z.number(),
    usageExceeded: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getVerificationsDaily = tb.buildPipe({
  pipe: "get_verifications_daily__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    keyId: z.string().optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    success: z.number(),
    rateLimited: z.number(),
    usageExceeded: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getVerificationsHourly = tb.buildPipe({
  pipe: "get_verifications_hourly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    keyId: z.string().optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: datetimeToUnixMilli,
    success: z.number(),
    rateLimited: z.number(),
    usageExceeded: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getActiveKeysHourly = tb.buildPipe({
  pipe: "get_active_keys_hourly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: datetimeToUnixMilli,
    keys: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getActiveKeysDaily = tb.buildPipe({
  pipe: "get_active_keys_daily__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    keys: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getActiveKeysWeekly = tb.buildPipe({
  pipe: "get_active_keys_weekly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    keys: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getActiveKeysMonthly = tb.buildPipe({
  pipe: "get_active_keys_monthly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    keys: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

/**
 * Across the entire time period
 */
export const getActiveKeys = tb.buildPipe({
  pipe: "get_active_keys__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    keys: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getQ1ActiveWorkspaces = tb.buildPipe({
  pipe: "get_q1_goal_distinct_workspaces__v1",
  parameters: z.object({}),
  data: z.object({
    workspaces: z.number(),
    time: datetimeToUnixMilli,
  }),
  opts: {
    cache: "no-store",
  },
});
