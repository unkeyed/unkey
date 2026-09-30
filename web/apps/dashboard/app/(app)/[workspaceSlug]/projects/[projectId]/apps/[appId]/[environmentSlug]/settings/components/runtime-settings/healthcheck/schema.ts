import { z } from "zod";

// TODO: extend when API supports more methods
export const HTTP_METHODS = ["GET", "POST"] as const;

export const INTERVAL_SECONDS = { min: 1, max: 3600, default: 30 } as const;

const PATH_PATTERN = /^\/[\w\-./]*$/;

const intervalMessage = `Use ${INTERVAL_SECONDS.min} to ${INTERVAL_SECONDS.max} seconds`;

export const healthcheckSchema = z.object({
  method: z.enum(HTTP_METHODS),
  path: z
    .string()
    .refine((path) => path === "" || path.startsWith("/"), "Path must start with /")
    .refine((path) => path === "" || PATH_PATTERN.test(path), "Invalid path characters"),
  intervalSeconds: z
    .number({ error: intervalMessage })
    .int(intervalMessage)
    .min(INTERVAL_SECONDS.min, intervalMessage)
    .max(INTERVAL_SECONDS.max, intervalMessage),
});

export type HealthcheckFormValues = z.infer<typeof healthcheckSchema>;
