import { relations } from "drizzle-orm";
import { bigint, datetime, index, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { keyAuth } from "./keyAuth";
import { webhooks } from "./webhooks";
import { workspaces } from "./workspaces";

export const usageReporters = mysqlTable(
  "usage_reporters",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    webhookId: varchar("webhook_id", { length: 256 })
      .notNull()
      .references(() => webhooks.id, { onDelete: "cascade" }),
    keySpaceId: varchar("key_space_id", { length: 256 })
      .notNull()
      .references(() => keyAuth.id, { onDelete: "cascade" }),
    workspaceId: varchar("workspace_id", { length: 256 })
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),

    // milliseconds
    interval: bigint("interval", { mode: "number" }).notNull(),
    // unix milli timestamp representing a time up to which all usage has been collected
    // new invocations must not process data prior to this watermark
    highWaterMark: bigint("high_water_mark", { mode: "number" }).notNull().default(0),
    nextExecution: bigint("next_execution", {
      mode: "number",
    }).notNull(),
    createdAt: datetime("created_at", { mode: "date", fsp: 3 }),
    deletedAt: datetime("deleted_at", { mode: "date", fsp: 3 }),
  },
  (table) => ({
    workspaceId: index("workspace_id_idx").on(table.workspaceId),
  }),
);

export const usageReportersRelations = relations(usageReporters, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [usageReporters.workspaceId],
    references: [workspaces.id],
  }),
  webhook: one(webhooks, {
    fields: [usageReporters.webhookId],
    references: [webhooks.id],
  }),
}));
