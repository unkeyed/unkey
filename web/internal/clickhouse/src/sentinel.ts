import { z } from "zod";
import type { Querier } from "./client";

export const sentinelLogsRequestSchema = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
  limit: z.number().int().positive().default(50),
  startTime: z.number().int(),
  endTime: z.number().int(),
});

export type SentinelLogsRequestSchema = z.infer<typeof sentinelLogsRequestSchema>;

export const sentinelLogsResponseSchema = z.object({
  request_id: z.string(),
  time: z.number().int(),
  deployment_id: z.string(),
  region: z.string(),
  method: z.string(),
  path: z.string(),
  response_status: z.number().int(),
  total_latency: z.number().int(),
  instance_latency: z.number().int(),
  sentinel_latency: z.number().int(),
});

export type SentinelLogsResponseSchema = z.infer<typeof sentinelLogsResponseSchema>;

export function getSentinelLogs(ch: Querier) {
  return async (args: SentinelLogsRequestSchema) => {
    const query = ch.query({
      query: `
        SELECT
          request_id,
          time,
          deployment_id,
          region,
          method,
          path,
          response_status,
          total_latency,
          instance_latency,
          sentinel_latency
        FROM default.sentinel_requests_raw_v1
        WHERE workspace_id = {workspaceId: String}
          AND project_id = {projectId: String}
          AND environment_id = {environmentId: String}
          AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
          AND deployment_id = {deploymentId: String}
        ORDER BY time DESC
        LIMIT {limit: Int}`,
      params: sentinelLogsRequestSchema,
      schema: sentinelLogsResponseSchema,
    });

    return query(args);
  };
}
