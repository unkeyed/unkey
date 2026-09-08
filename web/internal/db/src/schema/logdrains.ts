import { relations } from "drizzle-orm";
import {
  bigint,
  customType,
  index,
  int,
  mysqlEnum,
  mysqlTable,
  varchar,
} from "drizzle-orm/mysql-core";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

// logdrains configures continuous delivery of log streams to customer-owned
// HTTP or Axiom destinations.
export const logdrains = mysqlTable(
  "logdrains",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    name: varchar("name", { length: 128 }).notNull(),

    // Stream names are defined by svc/logdrain.
    stream: mysqlEnum("stream", ["audit_logs"]).notNull(),

    // Serialized logdrain.v1.Config. The destination stores its Vault
    // ciphertext next to the fields that use it.
    config: longblob("config").$type<Buffer>().notNull(),

    status: mysqlEnum("status", ["running", "paused_by_user", "paused_by_failure"])
      .notNull()
      .default("running"),
    consecutiveFailures: int("consecutive_failures").notNull().default(0),
    // This composite cursor identifies the last confirmed delivery. The event
    // ID uses bytewise ordering so events with the same millisecond are not lost.
    committedOffsetInsertedAt: bigint("committed_offset_inserted_at", { mode: "number" })
      .notNull()
      .default(0),
    committedOffsetEventId: customType<{ data: string }>({
      dataType() {
        return "varchar(255) COLLATE utf8mb4_0900_bin";
      },
    })("committed_offset_event_id")
      .notNull()
      .default(""),
    // Unix milliseconds before which the engine must not retry. Zero means due now.
    nextAttemptAt: bigint("next_attempt_at", { mode: "number" }).notNull().default(0),
    // One process shares this lease ID between its lease service and poller.
    // Each acquisition replaces the fencing token so stale workers cannot
    // change delivery state.
    leaseId: id("lease_id").notNull(),
    fencingToken: customType<{ data: string }>({
      dataType() {
        return "varchar(64) COLLATE utf8mb4_0900_as_cs";
      },
    })("fencing_token").notNull(),
    // Unix milliseconds computed from database time. Expired leases require a
    // new fencing token.
    leaseExpiresAt: bigint("lease_expires_at", { mode: "number" }).notNull().default(0),

    ...lifecycleDates,
  },
  (table) => [
    index("workspace_id_idx").on(table.workspaceId),
    index("lease_expires_at_id_idx").on(table.leaseExpiresAt, table.id),
    index("lease_id_status_next_attempt_at_id_idx").on(
      table.leaseId,
      table.status,
      table.nextAttemptAt,
      table.id,
    ),
  ],
);

export const logdrainsRelations = relations(logdrains, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [logdrains.workspaceId],
    references: [workspaces.id],
  }),
}));

export type SelectLogdrain = typeof logdrains.$inferSelect;
export type InsertLogdrain = typeof logdrains.$inferInsert;
