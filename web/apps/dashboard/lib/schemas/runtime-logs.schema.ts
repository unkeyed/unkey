import { z } from "zod";

export const runtimeLog = z.object({
  time: z.number().int(),
  severity: z.string(),
  message: z.string(),
  deployment_id: z.string(),
  k8s_pod_name: z.string(),
  region: z.string(),
  attributes: z.record(z.string(), z.unknown()).nullable(),
});

export type RuntimeLog = z.infer<typeof runtimeLog>;

export const runtimeLogsRequestSchema = z.object({
  projectId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
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

  podName: z
    .object({
      filters: z.array(
        z.object({
          operator: z.literal("is"),
          value: z.string(),
        }),
      ),
    })
    .nullable(),

  searchText: z.string().nullable(),
  cursor: z.number().nullable().optional(),
});

export type RuntimeLogsRequestSchema = z.infer<typeof runtimeLogsRequestSchema>;

export const runtimeLogsResponseSchema = z.object({
  logs: z.array(runtimeLog),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.int().optional(),
});

export type RuntimeLogsResponseSchema = z.infer<typeof runtimeLogsResponseSchema>;
