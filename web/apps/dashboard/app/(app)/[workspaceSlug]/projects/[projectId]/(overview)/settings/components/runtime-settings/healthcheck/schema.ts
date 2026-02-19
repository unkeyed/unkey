import { z } from "zod";

// TODO: extend when API supports more methods
export const HTTP_METHODS = ["GET", "POST"] as const;

export const INTERVAL_REGEX = /^\d+[smh]$/;

// TODO: MAX_CHECKS = 3 and array schema for multi-check when API supports
export const healthcheckSchema = z.object({
  method: z.enum(["GET", "POST"]),
  path: z
    .string()
    .min(1, "Path is required")
    .startsWith("/", "Path must start with /")
    .regex(/^\/[\w\-./]*$/, "Invalid path characters"),
  interval: z
    .string()
    .min(1, "Interval is required")
    .regex(INTERVAL_REGEX, "Use format like 15s, 2m, or 1h"),
});

export type HealthcheckFormValues = z.infer<typeof healthcheckSchema>;
