import { z } from "zod";
import type { Querier } from "./client";

export const sentinelRequest = z.object({
  workspaceId: z.string(),
  projectId: z.string(),
  deploymentId: z.string().optional(),
  limit: z.number().int().positive().default(50),
  startTime: z.number().int(),
  endTime: z.number().int(),
});

export type SentinelRequest = z.infer<typeof sentinelRequest>;

export const sentinelResponse = z.object({
  request_id: z.string(),
  time: z.number().int(),
  workspace_id: z.string(),
  environment_id: z.string(),
  project_id: z.string(),
  sentinel_id: z.string(),
  deployment_id: z.string(),
  region: z.string(),
  method: z.string(),
  host: z.string(),
  path: z.string(),
  query_string: z.string(),
  query_params: z.record(z.array(z.string())),
  request_headers: z.array(z.string()),
  request_body: z.string(),
  response_status: z.number().int(),
  response_headers: z.array(z.string()),
  response_body: z.string(),
  user_agent: z.string(),
  ip_address: z.string(),
  total_latency: z.number().int(),
  instance_latency: z.number().int(),
  sentinel_latency: z.number().int(),
});

export type SentinelResponse = z.infer<typeof sentinelResponse>;

export function getSentinelLogs(ch: Querier) {
  return async (args: SentinelRequest) => {
    const query = ch.query({
      query: `
        SELECT
          request_id,
          time,
          workspace_id,
          environment_id,
          project_id,
          sentinel_id,
          deployment_id,
          region,
          method,
          host,
          path,
          query_string,
          query_params,
          request_headers,
          request_body,
          response_status,
          response_headers,
          response_body,
          user_agent,
          ip_address,
          total_latency,
          instance_latency,
          sentinel_latency
        FROM default.sentinel_requests_raw_v1
        WHERE workspace_id = {workspaceId: String}
          AND time BETWEEN {startTime: Int64} AND {endTime: Int64}
          AND project_id = {projectId: String}
          ${args.deploymentId ? "AND deployment_id = {deploymentId: String}" : ""}
        ORDER BY time DESC
        LIMIT {limit: Int}`,
      params: sentinelRequest,
      schema: sentinelResponse,
    });

    return query(args);
  };
}
