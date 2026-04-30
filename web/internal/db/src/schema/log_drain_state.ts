import { relations } from "drizzle-orm";
import { bigint, int, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { logDrains } from "./log_drains";

// Hot-path state, written on every batch. Kept separate from log_drains so frequent
// UPDATEs do not bloat the config row's binlog entries or trigger config-change cache
// invalidations on the dashboard.
export const logDrainState = mysqlTable("log_drain_state", {
  drainId: varchar("drain_id", { length: 64 }).notNull().primaryKey(),

  lastDeliveryAt: bigint("last_delivery_at", { mode: "number" }),
  lastAttemptAt: bigint("last_attempt_at", { mode: "number" }),

  lastError: varchar("last_error", { length: 1024 }),
  consecutiveFailures: int("consecutive_failures").notNull().default(0),

  // Non-null when the drain is auto-paused. Set after consecutiveFailures crosses the
  // configured threshold. Cleared on resume.
  pausedReason: varchar("paused_reason", { length: 256 }),

  totalRecordsDelivered: bigint("total_records_delivered", { mode: "number" }).notNull().default(0),

  updatedAt: bigint("updated_at", { mode: "number" })
    .notNull()
    .$onUpdateFn(() => Date.now()),
});

export const logDrainStateRelations = relations(logDrainState, ({ one }) => ({
  drain: one(logDrains, {
    fields: [logDrainState.drainId],
    references: [logDrains.id],
  }),
}));
