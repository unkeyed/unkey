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
      AND bucket = {bucketId: String}
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

    // Targets are Nested(type, id, name, meta) — already grouped per
    // event_id in the table layout — so no GROUP BY / ARRAY JOIN
    // gymnastics needed. Map the four parallel arrays to row tuples
    // for the schema's [type, id, name, meta] shape.
    const logsQuery = ch.query({
      query: `
        SELECT
          event_id,
          time,
          event,
          description,
          actor_type,
          actor_id,
          actor_name,
          toJSONString(actor_meta) AS actor_meta,
          remote_ip,
          user_agent,
          toJSONString(meta) AS meta,
          arrayMap(
            (targetType, targetId, targetName, targetMeta) ->
              (targetType, targetId, targetName, toJSONString(targetMeta)),
            \`targets.type\`, \`targets.id\`, \`targets.name\`, \`targets.meta\`
          ) AS targets
        FROM ${TABLE}
        WHERE ${filterConditions}
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
