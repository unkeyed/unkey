import { bigint, index, int, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { primaryKey } from "./util/primary_key";

// backoffice_audit_outbox is the transactional outbox for the staff back
// office. Every staff edit inserts its audit event here in the same MySQL
// transaction as the edit, so no edit can exist without its audit record.
// The back office drains rows by pk order with FOR UPDATE SKIP LOCKED into
// ClickHouse `backoffice.audit_logs_v1` and stamps drained_at (no hard
// delete, same as clickhouse_outbox).
//
// Deliberately separate from clickhouse_outbox: that drainer exports to
// the customer-visible audit log, and staff actions must never appear
// there. The only thing in this repo that touches this table is the daily
// RunAuditLogOutboxCleanup cron, which hard-deletes drained rows after the
// same 30-day retention it applies to clickhouse_outbox.
export const backofficeAuditOutbox = mysqlTable(
  "backoffice_audit_outbox",
  {
    pk: primaryKey(),
    // boal_ id; also the ClickHouse dedupe key, so a re-drain is harmless.
    eventId: varchar("event_id", { length: 64 }).notNull().unique(),
    // The ClickHouse row, exactly as it will be inserted.
    payload: json("payload").notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    drainedAt: bigint("drained_at", { mode: "number", unsigned: true }),
    attempts: int("attempts", { unsigned: true }).notNull().default(0),
    lastError: varchar("last_error", { length: 512 }),
  },
  (table) => [
    // Drainer hot path: WHERE drained_at IS NULL ORDER BY pk.
    index("drainer_pending_idx").on(table.drainedAt, table.pk),
  ],
);
