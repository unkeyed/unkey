import { sentinelResponse } from "@unkey/clickhouse/src/sentinel";
import { z } from "zod";

export type SentinelLogsRequestSchema = z.infer<typeof sentinelLogsRequestSchema>;
export const sentinelLogsRequestSchema = z.object({
  projectId: z.string(),
  deploymentId: z.string(),
  limit: z.number().int().min(1).max(100).default(50),
  startTime: z.number().int(),
  endTime: z.number().int(),
});

export type SentinelLogsResponseSchema = z.infer<typeof sentinelLogsResponseSchema>;
export const sentinelLogsResponseSchema = z.array(sentinelResponse);
