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
  appId: z.string().nullable().optional(),
  deploymentId: z.array(z.string()).optional().default([]),
  environmentId: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  limit: z.int().min(1).max(1_000),
  startTime: z.int().nullable(),
  endTime: z.int().nullable(),
  since: z.string().nullable(),
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
  // 1-based page for offset pagination. Defaults to 1 (offset 0).
  page: z.number().int().min(1).optional().default(1),
  // Gates the count(*) query. Single-page callers (e.g. the live poll) pass
  // false to skip the extra ClickHouse round-trip; paginated callers need the
  // total to render page counts.
  includeTotal: z.boolean().optional().default(true),
});

export type RuntimeLogsRequestSchema = z.infer<typeof runtimeLogsRequestSchema>;

export const runtimeLogsResponseSchema = z.object({
  logs: z.array(dashboardRuntimeLog),
  total: z.number().int(),
});

export type RuntimeLogsResponseSchema = z.infer<typeof runtimeLogsResponseSchema>;
