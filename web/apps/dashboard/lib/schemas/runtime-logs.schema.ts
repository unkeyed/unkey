import { runtimeLog } from "@unkey/clickhouse/src/runtime-logs";
import { z } from "zod";

export const dashboardRuntimeLog = runtimeLog.omit({ k8s_pod_name: true }).extend({
  instance_id: z.string(),
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
  total: z.number(),
  nextCursor: z.int().optional(),
});

export type RuntimeLogsResponseSchema = z.infer<typeof runtimeLogsResponseSchema>;
