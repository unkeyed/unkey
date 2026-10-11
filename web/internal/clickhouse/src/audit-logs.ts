import { Err, Ok, type Result } from "@unkey/error";
import { z } from "zod";
import type { QueryError } from "./client/error";
import type { Querier } from "./client/interface";

const TABLE = "default.audit_logs_raw_v1";
const EMPTY_JSON = "{}";

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

const auditLogJsonLookupSchema = z.object({
  workspaceId: z.string(),
  bucketId: z.string(),
  times: z.array(z.int()),
  eventIds: z.array(z.string()),
});

// workspace_id and bucket are intentionally omitted: they're always equal
// to the query filter values and selecting `any(workspace_id) AS workspaceId`
// would shadow the WHERE column and trigger ILLEGAL_AGGREGATION in ClickHouse.
// Callers reconstruct those fields from their own context.
export const auditLogRow = z.object({
  eventId: z.string(),
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
      "bucket = {bucketId: String}",
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
          AND bucket = {bucketId: String}
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
      getLogsQuery: (): Promise<Result<AuditLogRow[], QueryError>> =>
        loadAuditLogPage(pageQuery, jsonQuery, args),
      getTotalQuery: () => totalQuery(args),
    };
  };
}

async function loadAuditLogPage(
  pageQuery: (args: AuditLogsRequest) => Promise<Result<AuditLogPageRow[], QueryError>>,
  jsonQuery: (args: {
    workspaceId: string;
    bucketId: string;
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
    bucketId: args.bucketId,
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
