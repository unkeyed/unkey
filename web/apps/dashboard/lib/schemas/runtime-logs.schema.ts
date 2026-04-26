import { z } from "zod";

export const dashboardRuntimeLog = z.object({
  time: z.int(),
  severity: z.string(),
  message: z.string(),
  deployment_id: z.string(),
  region: z.string(),
  instance_id: z.string(),
  attributes: z.record(z.string(), z.unknown()).nullable(),
});

export type RuntimeLog = z.infer<typeof dashboardRuntimeLog>;

export const runtimeLogsRequestSchema = z.object({
  projectId: z.string(),
  deploymentId: z.string().nullable().optional(),
  environmentId: z.string().nullable().optional(),
  limit: z.int(),
  startTime: z.int(),
  endTime: z.int(),
  since: z.string(),
  severity: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  region: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  message: z.string().nullable(),
  instanceId: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  cursor: z.number().nullable().optional(),
});

export type RuntimeLogsRequestSchema = z.infer<typeof runtimeLogsRequestSchema>;

export const runtimeLogsResponseSchema = z.object({
  logs: z.array(dashboardRuntimeLog),
  hasMore: z.boolean(),
  nextCursor: z.int().optional(),
});

export type RuntimeLogsResponseSchema = z.infer<typeof runtimeLogsResponseSchema>;
