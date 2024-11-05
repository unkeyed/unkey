import { z } from "zod";
import type { Querier } from "./client";

export function getLogs(ch: Querier) {
  return async (args: { workspaceId: string; limit: number }) => {
    const query = ch.query({
      query: `
    SELECT
      request_id,
      time,
      workspace_id,
      host,
      method,
      path,
      request_headers,
      request_body,
      response_status,
      response_headers,
      response_body,
      error,
      service_latency
    FROM default.raw_api_requests_v1
    WHERE workspace_id = {workspaceId: String}
    ORDER BY time DESC
    LIMIT {limit: Int}`,
      params: z.object({
        workspaceId: z.string(),
        limit: z.number().int(),
      }),
      schema: z.object({
        request_id: z.string(),
        time: z.number().int(),
        workspace_id: z.string(),
        host: z.string(),
        method: z.string(),
        path: z.string(),
        request_headers: z.array(z.string()),
        request_body: z.string(),
        response_status: z.number().int(),
        response_headers: z.array(z.string()),
        response_body: z.string(),
        error: z.string(),
        service_latency: z.number().int(),
      }),
    });

    return query(args);
  };
}
