import { clickhouse } from "@/lib/clickhouse";
import { z } from "zod";

export const getLogsClickhousePayload = z.object({
  workspaceId: z.string(),
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  path: z.string().optional().nullable(),
  host: z.string().optional().nullable(),
  requestId: z.string().optional().nullable(),
  method: z.string().optional().nullable(),
  responseStatus: z.array(z.number().int()).nullable(),
});
type GetLogsClickhousePayload = z.infer<typeof getLogsClickhousePayload>;

export async function getLogs(args: GetLogsClickhousePayload) {
  const query = clickhouse.querier.query({
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
    FROM metrics.raw_api_requests_v1
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
                WHEN {responseStatus: Array(UInt16)} IS NOT NULL AND length({responseStatus: Array(UInt16)}) > 0 THEN
                    response_status IN (
                        SELECT status
                        FROM (
                            SELECT 
                                multiIf(
                                    code = 200, arrayJoin(range(200, 300)),
                                    code = 400, arrayJoin(range(400, 500)),
                                    code = 500, arrayJoin(range(500, 600)),
                                    code
                                ) as status
                            FROM (
                                SELECT arrayJoin({responseStatus: Array(UInt16)}) as code
                            )
                        )
                    )
                ELSE TRUE
            END)
    ORDER BY time DESC
    LIMIT {limit: Int}`,
    params: getLogsClickhousePayload,
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
