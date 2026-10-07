import { z } from "zod";
import type { Querier } from "./client/interface";

const TABLE = "default.audit_logs_raw_v1";

export const auditLogsRequestSchema = z.object({
  workspaceId: z.string(),
  buckets: z.array(z.string()).min(1),
  limit: z.int(),
  offset: z.int(),
  startTime: z.int(),
  endTime: z.int(),
  events: z.array(z.string()),
  actorIds: z.array(z.string()),
});

export type AuditLogsRequest = z.infer<typeof auditLogsRequestSchema>;

// workspace_id is omitted because it always equals the filter value. bucket
// is returned because a caller may read several buckets in one query.
export const auditLogRow = z.object({
  eventId: z.string(),
  bucket: z.string(),
  time: z.int(),
  event: z.string(),
  description: z.string(),
  actorType: z.string(),
  actorId: z.string(),
  actorName: z.string(),
  actorMeta: z.string(),
  remoteIp: z.string(),
  userAgent: z.string(),
  meta: z.string(),
  targets: z.array(z.tuple([z.string(), z.string(), z.string(), z.string()])),
});

export type AuditLogRow = z.infer<typeof auditLogRow>;

export function getAuditLogs(ch: Querier) {
  return (args: AuditLogsRequest) => {
    // Conditions are built per-call rather than wrapped in CASE WHEN so CH
    // can push event/actor predicates into the set/bloom skip indexes.
    const conditions = [
      "workspace_id = {workspaceId: String}",
      "bucket IN {buckets: Array(String)}",
      "time BETWEEN {startTime: UInt64} AND {endTime: UInt64}",
    ];
    if (args.events.length > 0) {
      conditions.push("event IN {events: Array(String)}");
    }
    if (args.actorIds.length > 0) {
      conditions.push("actor_id IN {actorIds: Array(String)}");
    }
    const filterConditions = conditions.join(" AND ");

    // Targets are Nested(type, id, name, meta) — already grouped per
    // event_id in the table layout — so no GROUP BY / ARRAY JOIN
    // gymnastics needed. Map the four parallel arrays to row tuples
    // for the schema's [type, id, name, meta] shape.
    const logsQuery = ch.query({
      query: `
        SELECT
          event_id AS eventId,
          bucket,
          time,
          event,
          description,
          actor_type AS actorType,
          actor_id AS actorId,
          actor_name AS actorName,
          toJSONString(actor_meta) AS actorMeta,
          remote_ip AS remoteIp,
          user_agent AS userAgent,
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

    // count() over countDistinct(event_id): event_id is row-identity and the
    // ReplacingMergeTree dedup catches retries, so the only divergence is a
    // few unmerged rows in a short window. The result is cached for 5min
    // upstream, which makes that drift invisible.
    const totalQuery = ch.query({
      query: `
        SELECT count() AS totalCount
        FROM ${TABLE}
        WHERE ${filterConditions}`,
      params: auditLogsRequestSchema,
      schema: z.object({ totalCount: z.int() }),
    });

    return {
      getLogsQuery: () => logsQuery(args),
      getTotalQuery: () => totalQuery(args),
    };
  };
}
