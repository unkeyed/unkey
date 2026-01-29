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

export const sentinelRpsRequestSchema = z.object({
  workspaceId: z.string(),
  deploymentId: z.string(),
  startTime: z.number().int(),
});

export const sentinelRpsResponseSchema = z.object({
  region: z.string(),
  avg_rps: z.number(),
});

export const instanceRpsRequestSchema = z.object({
  workspaceId: z.string(),
  deploymentId: z.string(),
  startTime: z.number().int(),
});

export const instanceRpsResponseSchema = z.object({
  instance_id: z.string(),
  avg_rps: z.number(),
});

export function getSentinelRps(ch: Querier) {
  return async (args: z.infer<typeof sentinelRpsRequestSchema>) => {
    const query = ch.query({
      query: `
        SELECT
          region,
          round(COUNT(*) * 1000.0 / (toUnixTimestamp(now()) * 1000 - {startTime: UInt64}), 2) as avg_rps
        FROM default.sentinel_requests_raw_v1
        WHERE workspace_id = {workspaceId: String}
          AND deployment_id = {deploymentId: String}
          AND time >= {startTime: UInt64}
        GROUP BY region`,
      params: sentinelRpsRequestSchema,
      schema: sentinelRpsResponseSchema,
    });

    return query(args);
  };
}

export function getInstanceRps(ch: Querier) {
  return async (args: z.infer<typeof instanceRpsRequestSchema>) => {
    const query = ch.query({
      query: `
        SELECT
          instance_id,
          round(COUNT(*) * 1000.0 / (toUnixTimestamp(now()) * 1000 - {startTime: UInt64}), 2) as avg_rps
        FROM default.sentinel_requests_raw_v1
        WHERE workspace_id = {workspaceId: String}
          AND deployment_id = {deploymentId: String}
          AND time >= {startTime: UInt64}
        GROUP BY instance_id`,
      params: instanceRpsRequestSchema,
      schema: instanceRpsResponseSchema,
    });

    return query(args);
  };
}
