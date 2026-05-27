import { bigint, boolean, index, mysqlTable, varchar } from "drizzle-orm/mysql-core";

// build_slots holds one row per deployment currently consuming a build
// concurrency slot for its workspace. Capacity is enforced by counting rows
// joined against deployments.status — any row whose deployment has gone
// terminal is automatically excluded from the count, so leaked rows are
// inert rather than blocking.
export const buildSlots = mysqlTable(
  "build_slots",
  {
    deploymentId: varchar("deployment_id", { length: 255 }).notNull().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    acquiredAt: bigint("acquired_at", { mode: "number" }).notNull(),
  },
  (table) => [index("idx_workspace").on(table.workspaceId)],
);

// build_slot_waiters parks deployments that hit the workspace's concurrency
// cap. ON DUPLICATE KEY updates the awakeable on re-entry so a retrying Deploy
// handler always has a fresh awakeable registered.
export const buildSlotWaiters = mysqlTable(
  "build_slot_waiters",
  {
    deploymentId: varchar("deployment_id", { length: 255 }).notNull().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    awakeableId: varchar("awakeable_id", { length: 255 }).notNull(),
    isProduction: boolean("is_production").notNull(),
    enqueuedAt: bigint("enqueued_at", { mode: "number" }).notNull(),
  },
  (table) => [
    index("idx_workspace_priority").on(table.workspaceId, table.isProduction, table.enqueuedAt),
  ],
);
