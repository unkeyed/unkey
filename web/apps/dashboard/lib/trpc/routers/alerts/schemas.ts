import { alertMetrics } from "@unkey/db/src/schema";
import { z } from "zod";

export const alertMetricSchema = z.enum(alertMetrics);

export const listAlertsInput = z.object({
  status: z.enum(["open", "resolved", "all"]).default("open"),
  metric: alertMetricSchema.optional(),
  appId: z.string().min(1).optional(),
  cursor: z.string().min(1).optional(),
  limit: z.number().int().min(1).max(100).default(50),
});

export const getAlertInput = z.object({
  alertId: z.string().min(1),
});

export const resolveAlertInput = z.object({
  alertId: z.string().min(1),
  message: z.string().trim().min(1).max(1000),
});
