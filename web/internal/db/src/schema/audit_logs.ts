import { newId } from "@unkey/id";
import { relations } from "drizzle-orm";
import { bigint, index, json, mysqlTable, unique, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const auditLog = mysqlTable(
  "audit_log",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 })
      .notNull()
      .unique()

      .$defaultFn(() => newId("auditLog")),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    // bucket is the name of the bucket that the target belongs to
    bucket: varchar("bucket", { length: 256 }).notNull().default("unkey_mutations"),
    // @deprecated
    bucketId: varchar("bucket_id", { length: 256 }).notNull(),
    event: varchar("event", { length: 256 }).notNull(),

    // When the event happened
    time: bigint("time", { mode: "number" })
      .notNull()
      .$defaultFn(() => Date.now()),
    // A human readable description of the event
    display: varchar("display", { length: 256 }).notNull(),

    remoteIp: varchar("remote_ip", { length: 256 }),
    userAgent: varchar("user_agent", { length: 256 }),
    actorType: varchar("actor_type", { length: 256 }).notNull(),
    actorId: varchar("actor_id", { length: 256 }).notNull(),
    actorName: varchar("actor_name", { length: 256 }),
    actorMeta: json("actor_meta"),

    ...lifecycleDates,
  },
  (table) => [
    // Every dashboard SELECT filters by (workspace_id, bucket, time) and orders
    // by time DESC. This one composite serves every code path; the old
    // single-column indexes on workspace_id / bucket / bucket_id / event /
    // actor_id / time saw zero traffic and were costing INSERT throughput.
    index("workspace_id_bucket_time_idx").on(table.workspaceId, table.bucket, table.time),
  ],
);

export const auditLogRelations = relations(auditLog, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [auditLog.workspaceId],
    references: [workspaces.id],
  }),
  // bucket: one(auditLogBucket, {
  //   fields: [auditLog.bucketId],
  //   references: [auditLogBucket.id],
  // }),
  targets: many(auditLogTarget),
}));

export const auditLogTarget = mysqlTable(
  "audit_log_target",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    // @deprecated
    bucketId: varchar("bucket_id", { length: 256 }).notNull(),

    // bucket is the name of the bucket that the target belongs to
    bucket: varchar("bucket", { length: 256 }).notNull().default("unkey_mutations"),
    auditLogId: varchar("audit_log_id", { length: 256 }).notNull(),

    // A human readable name to display in the UI
    displayName: varchar("display_name", { length: 256 }).notNull(),
    // the type of the target
    type: varchar("type", { length: 256 }).notNull(),
    // the id of the target
    id: varchar("id", { length: 256 }).notNull(),
    // the name of the target
    name: varchar("name", { length: 256 }),
    // the metadata of the target
    meta: json("meta"),

    ...lifecycleDates,
  },
  (table) => [
    unique("unique_id_per_log").on(table.auditLogId, table.id),
    index("id_idx").on(table.id),
  ],
);

export const auditLogTargetRelations = relations(auditLogTarget, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [auditLogTarget.workspaceId],
    references: [workspaces.id],
  }),
  // bucket: one(auditLogBucket, {
  //   fields: [auditLogTarget.bucketId],
  //   references: [auditLogBucket.id],
  // }),
  log: one(auditLog, {
    fields: [auditLogTarget.auditLogId],
    references: [auditLog.id],
  }),
}));

export type SelectAuditLog = typeof auditLog.$inferSelect;
export type SelectAuditLogTarget = typeof auditLogTarget.$inferSelect;

// clickhouse_outbox is the transactional outbox for ClickHouse export.
// Writers insert one row per logical event in the same MySQL transaction
// as the underlying mutation; the AuditLogExportService worker drains
// rows by pk order, ships them to ClickHouse, and DELETEs them on
// success. Steady state is "minutes of backlog" — if it grows, the
// drainer is wedged.
//
// The table is intentionally generic. Today only audit logs write to it
// (version = "audit_log.v1") but it's named clickhouse_outbox (not
// audit_log_outbox) so other producers can share it later without
// another migration.
//
// version is a namespaced schema tag like "audit_log.v1". The drainer
// filters `WHERE version IN (known)` so an old drainer never chokes on
// a newer payload shape; unknown-version rows pile up until a drainer
// with the matching handler ships. Use the version prefix to name the
// producer (`audit_log.*`, `customer.audit_log.*`, `key_lifecycle.*`)
// so producers/consumers don't collide.
//
// payload is the JSON-encoded event envelope (actor, targets, meta).
// workspace_id and event_id are duplicated out of the payload so the
// drainer can route + dedupe without decoding, and so on-call can grep
// the table during incidents.
//
// deleted_at is stamped (millis) when the drainer confirms the CH insert
// instead of hard-deleting. Lets ops re-queue events by clearing
// deleted_at, and lets us keep an audit trail of what was exported. No
// sweep job for now; revisit when storage matters.
export const clickhouseOutbox = mysqlTable(
  "clickhouse_outbox",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    version: varchar("version", { length: 64 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    eventId: varchar("event_id", { length: 256 }).notNull(),
    payload: json("payload").notNull(),
    createdAt: bigint("created_at", { mode: "number" })
      .notNull()
      .$defaultFn(() => Date.now()),
    deletedAt: bigint("deleted_at", { mode: "number", unsigned: true }),
  },
  (table) => [
    // Drainer hot path: WHERE version IN (...) AND deleted_at IS NULL
    // ORDER BY pk FOR UPDATE SKIP LOCKED. Without this index the table
    // scan walks every marked row (deleted_at IS NOT NULL) before
    // finding the next unmarked one — eventually 99% wasted IO since
    // marked rows accumulate forever (no sweep). Leading deleted_at
    // lets MySQL seek directly to the NULL bucket; pk last keeps the
    // ORDER BY satisfied without a sort.
    index("drainer_pending_idx").on(table.deletedAt, table.version, table.pk),
  ],
);

export type SelectClickhouseOutbox = typeof clickhouseOutbox.$inferSelect;
