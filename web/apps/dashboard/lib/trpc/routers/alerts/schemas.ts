import { alertMetrics } from "@unkey/db/src/schema";
import { z } from "zod";

export const alertMetricSchema = z.enum(alertMetrics);
export const alertSeriesMetricSchema = z.enum([
  "error_5xx",
  "error_4xx",
  "requests",
  "egress_bytes",
  "cpu_seconds",
  "memory_utilization",
  "health",
]);

const timeRange = z
  .object({
    startMs: z.number().int().nonnegative(),
    endMs: z.number().int().nonnegative(),
  })
  .refine(({ startMs, endMs }) => startMs < endMs, {
    message: "startMs must be before endMs",
  });

export const listAlertsInput = z
  .object({
    includeResolved: z.boolean().default(false),
    metric: alertMetricSchema.optional(),
    appId: z.string().min(1).optional(),
    environmentId: z.string().min(1).optional(),
    startMs: z.number().int().nonnegative().optional(),
    endMs: z.number().int().nonnegative().optional(),
    cursor: z.string().min(1).optional(),
    limit: z.number().int().min(1).max(100).default(50),
  })
  .refine(({ startMs, endMs }) => startMs === undefined || endMs === undefined || startMs < endMs, {
    message: "startMs must be before endMs",
  });

export const getAlertInput = z.object({
  alertId: z.string().min(1),
});

const alertSeriesMaximumRangeMs = 8 * 24 * 60 * 60 * 1000;

export const alertSeriesInput = timeRange
  .extend({
    appId: z.string().min(1),
    environmentId: z.string().min(1),
    metric: alertSeriesMetricSchema,
    resolution: z.enum(["5m", "1h"]).default("5m"),
  })
  .refine(({ startMs, endMs }) => endMs - startMs <= alertSeriesMaximumRangeMs, {
    message: "Series range must not exceed 8 days",
  });

export const alertDeploymentsInput = timeRange.extend({
  appId: z.string().min(1),
  environmentId: z.string().min(1),
});
