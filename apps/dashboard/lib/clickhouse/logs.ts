import { type Clickhouse, Client, Noop } from "@unkey/clickhouse-zod";
import { z } from "zod";
import { env } from "../env";

// dummy example of how to query stuff from clickhouse
export async function getLogs(args: {
  workspaceId: string;
  limit: number;
  startTime: number;
  endTime: number;
  path?: string | null;
  host?: string | null;
  requestId?: string | null;
  method?: string | null;
  response_status: number | null;
}) {
  const { CLICKHOUSE_URL } = env();
  const ch: Clickhouse = CLICKHOUSE_URL ? new Client({ url: CLICKHOUSE_URL }) : new Noop();
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
        AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
        AND (CASE
                WHEN {host: String} != '' THEN host = {host: String}
                ELSE TRUE
            END)
        AND (CASE
                WHEN {requestId: String} != '' THEN request_id = {requestId: String}
                ELSE TRUE
            END)
        AND (CASE
                WHEN {method: String} != '' THEN method = {method: String}
                ELSE TRUE
            END)
        AND (CASE
                WHEN {path: String} != '' THEN path = {path: String}
                ELSE TRUE
            END)
       AND (CASE
              WHEN {response_status: Nullable(UInt16)} IS NOT NULL THEN
                  CASE
                    WHEN {response_status: UInt16} = 200 THEN response_status >= 200 AND response_status < 300
                    WHEN {response_status: UInt16} = 400 THEN response_status >= 400 AND response_status < 500
                    WHEN {response_status: UInt16} = 500 THEN response_status >= 500
                    WHEN {response_status: Int16} = 0 THEN TRUE
                    ELSE FALSE
                  END
              ELSE TRUE
          END)
    ORDER BY time DESC
    LIMIT {limit: Int}`,
    params: z.object({
      workspaceId: z.string(),
      limit: z.number().int(),
      startTime: z.number().int(),
      endTime: z.number().int(),
      path: z.string().optional().nullable(),
      host: z.string().optional().nullable(),
      requestId: z.string().optional().nullable(),
      method: z.string().optional().nullable(),
      response_status: z.number().int().nullable(),
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
}
