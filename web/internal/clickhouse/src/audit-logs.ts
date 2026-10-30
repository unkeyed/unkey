import { Err, Ok, type Result } from "@unkey/error";
import { z } from "zod";
import type { QueryError } from "./client/error";
import type { Querier } from "./client/interface";

const TABLE = "default.audit_logs_raw_v1";
const EMPTY_JSON = "{}";

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

const auditLogJsonLookupSchema = z.object({
  workspaceId: z.string(),
  buckets: z.array(z.string()).min(1),
  times: z.array(z.int()),
  eventIds: z.array(z.string()),
});

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

const auditLogPageRow = auditLogRow.omit({
  actorMeta: true,
  meta: true,
  targets: true,
});

type AuditLogPageRow = z.infer<typeof auditLogPageRow>;

const auditLogJsonRow = auditLogRow.pick({
  eventId: true,
  time: true,
  actorMeta: true,
  meta: true,
  targets: true,
});

type AuditLogJsonRow = z.infer<typeof auditLogJsonRow>;

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

    // No JSON columns here. ORDER BY matches the primary key after the
    // workspace and bucket equalities, so read-in-order stops after LIMIT
    // rows instead of sorting the retention window.
    const pageQuery = ch.query({
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
          remote_ip AS remoteIp,
          user_agent AS userAgent
        FROM ${TABLE}
        WHERE ${filterConditions}
        ORDER BY time DESC, event_id DESC
        LIMIT {limit: Int} OFFSET {offset: Int}
        SETTINGS optimize_read_in_order = 1`,
      params: auditLogsRequestSchema,
      schema: auditLogPageRow,
    });

    // Targets are Nested(type, id, name, meta), already grouped per event.
    // time is the third primary-key column, so this lookup reads the page's
    // granules. The pair match drops other events that share those timestamps.
    const jsonQuery = ch.query({
      query: `
        SELECT
          event_id AS eventId,
          time,
          toJSONString(actor_meta) AS actorMeta,
          toJSONString(meta) AS meta,
          arrayMap(
            (targetType, targetId, targetName, targetMeta) ->
              (targetType, targetId, targetName, toJSONString(targetMeta)),
            \`targets.type\`, \`targets.id\`, \`targets.name\`, \`targets.meta\`
          ) AS targets
        FROM ${TABLE}
        WHERE workspace_id = {workspaceId: String}
          AND bucket IN {buckets: Array(String)}
          AND time IN {times: Array(Int64)}
          AND (time, event_id) IN (
            SELECT
              tupleElement(pair, 1),
              tupleElement(pair, 2)
            FROM (
              SELECT arrayJoin(arrayZip({times: Array(Int64)}, {eventIds: Array(String)})) AS pair
            )
          )`,
      params: auditLogJsonLookupSchema,
      schema: auditLogJsonRow,
    });

    return {
      getLogsQuery: (): Promise<Result<AuditLogRow[], QueryError>> =>
        loadAuditLogPage(pageQuery, jsonQuery, args),
    };
  };
}

export const latestAuditLogByTargetRequestSchema = z.object({
  workspaceId: z.string(),
  bucket: z.string(),
  event: z.string(),
  targetId: z.string(),
});

export type LatestAuditLogByTargetRequest = z.infer<typeof latestAuditLogByTargetRequestSchema>;

export const latestAuditLogByTargetRow = auditLogRow
  .pick({ time: true, actorType: true, actorId: true, actorName: true })
  .extend({ targets: z.array(z.tuple([z.string(), z.string()])) });

export type LatestAuditLogByTargetRow = z.infer<typeof latestAuditLogByTargetRow>;

export function getLatestAuditLogByTarget(ch: Querier) {
  return ch.query({
    query: `
      SELECT
        time,
        actor_type AS actorType,
        actor_id AS actorId,
        actor_name AS actorName,
        arrayZip(\`targets.type\`, \`targets.id\`) AS targets
      FROM ${TABLE}
      WHERE workspace_id = {workspaceId: String}
        AND bucket = {bucket: String}
        AND event = {event: String}
        AND has(\`targets.id\`, {targetId: String})
      ORDER BY time DESC, event_id DESC
      LIMIT 1
      SETTINGS optimize_read_in_order = 1`,
    params: latestAuditLogByTargetRequestSchema,
    schema: latestAuditLogByTargetRow,
  });
}

async function loadAuditLogPage(
  pageQuery: (args: AuditLogsRequest) => Promise<Result<AuditLogPageRow[], QueryError>>,
  jsonQuery: (args: {
    workspaceId: string;
    buckets: string[];
    times: number[];
    eventIds: string[];
  }) => Promise<Result<AuditLogJsonRow[], QueryError>>,
  args: AuditLogsRequest,
): Promise<Result<AuditLogRow[], QueryError>> {
  const page = await pageQuery(args);
  if (page.err) {
    return Err(page.err);
  }
  if (page.val.length === 0) {
    return Ok([]);
  }

  const json = await jsonQuery({
    workspaceId: args.workspaceId,
    buckets: args.buckets,
    times: page.val.map((row) => row.time),
    eventIds: page.val.map((row) => row.eventId),
  });
  if (json.err) {
    return Err(json.err);
  }

  const byTime = new Map<number, Map<string, AuditLogJsonRow>>();
  for (const row of json.val) {
    let byId = byTime.get(row.time);
    if (!byId) {
      byId = new Map();
      byTime.set(row.time, byId);
    }
    if (!byId.has(row.eventId)) {
      byId.set(row.eventId, row);
    }
  }

  return Ok(
    page.val.map((row) => {
      const jsonRow = byTime.get(row.time)?.get(row.eventId);
      return {
        ...row,
        actorMeta: jsonRow?.actorMeta ?? EMPTY_JSON,
        meta: jsonRow?.meta ?? EMPTY_JSON,
        targets: jsonRow?.targets ?? [],
      };
    }),
  );
}
