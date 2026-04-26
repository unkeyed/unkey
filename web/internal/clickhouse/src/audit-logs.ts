import { z } from "zod";
import type { Querier } from "./client/interface";

const TABLE = "default.audit_logs_raw_v1";

export const auditLogsRequestSchema = z.object({
  workspaceId: z.string(),
  bucketId: z.string(),
  limit: z.int(),
  offset: z.int(),
  startTime: z.int(),
  endTime: z.int(),
  events: z.array(z.string()),
  actorIds: z.array(z.string()),
});

export type AuditLogsRequest = z.infer<typeof auditLogsRequestSchema>;

// workspace_id and bucket_id are intentionally omitted: they're always equal
// to the query filter values and selecting `any(workspace_id) AS workspace_id`
// would shadow the WHERE column and trigger ILLEGAL_AGGREGATION in ClickHouse.
// Callers reconstruct those fields from their own context.
export const auditLogRow = z.object({
  event_id: z.string(),
  time: z.int(),
  event: z.string(),
  description: z.string(),
  actor_type: z.string(),
  actor_id: z.string(),
  actor_name: z.string(),
  actor_meta: z.string(),
  remote_ip: z.string(),
  user_agent: z.string(),
  meta: z.string(),
  targets: z.array(z.tuple([z.string(), z.string(), z.string(), z.string()])),
});

export type AuditLogRow = z.infer<typeof auditLogRow>;

export function getAuditLogs(ch: Querier) {
  return (args: AuditLogsRequest) => {
    const filterConditions = `
      workspace_id = {workspaceId: String}
      AND bucket_id = {bucketId: String}
      AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
      AND (
        CASE
          WHEN length({events: Array(String)}) > 0 THEN event IN {events: Array(String)}
          ELSE TRUE
        END
      )
      AND (
        CASE
          WHEN length({actorIds: Array(String)}) > 0 THEN actor_id IN {actorIds: Array(String)}
          ELSE TRUE
        END
      )
    `;

    // Filter in a subquery so the outer aggregates can alias back to the
    // original column names without triggering ClickHouse's
    // ILLEGAL_AGGREGATION guard (which rejects `any(col) AS col` when `col`
    // also appears in WHERE).
    const logsQuery = ch.query({
      query: `
        SELECT
          event_id,
          any(time) AS time,
          any(event) AS event,
          any(description) AS description,
          any(actor_type) AS actor_type,
          any(actor_id) AS actor_id,
          any(actor_name) AS actor_name,
          any(actor_meta) AS actor_meta,
          any(remote_ip) AS remote_ip,
          any(user_agent) AS user_agent,
          any(meta) AS meta,
          groupUniqArray((target_type, target_id, target_name, target_meta)) AS targets
        FROM (
          SELECT *
          FROM ${TABLE}
          WHERE ${filterConditions}
        )
        GROUP BY event_id
        ORDER BY time DESC, event_id DESC
        LIMIT {limit: Int} OFFSET {offset: Int}`,
      params: auditLogsRequestSchema,
      schema: auditLogRow,
    });

    const totalQuery = ch.query({
      query: `
        SELECT countDistinct(event_id) AS total_count
        FROM ${TABLE}
        WHERE ${filterConditions}`,
      params: auditLogsRequestSchema,
      schema: z.object({ total_count: z.int() }),
    });

    return {
      logsQuery: logsQuery(args),
      totalQuery: totalQuery(args),
    };
  };
}
