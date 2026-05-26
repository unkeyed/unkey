import { bigint, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";

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
