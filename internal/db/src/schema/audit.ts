import { relations } from "drizzle-orm";
import {
  datetime,
  index,
  json,
  mysqlEnum,
  mysqlTable,
  primaryKey,
  varchar,
} from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { keys } from "./keys";
import { workspaces } from "./workspaces";

export const auditLogs = mysqlTable(
  "audit_logs",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    // under what workspace this happened
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    /**
     * A machine readable description of what happened
     */
    action: mysqlEnum("action", ["create", "update", "delete"]).notNull(),

    /**
     * A human readable description of what happened.
     */
    description: varchar("description", { length: 512 }).notNull(),
    time: datetime("time", { fsp: 3 }).notNull(), // unix milli
    actorType: mysqlEnum("actor_type", ["user", "key"]).notNull(),
    actorId: varchar("actor_id", { length: 256 }).notNull(),
    resourceType: mysqlEnum("resource_type", [
      "key",
      "api",
      "workspace",
      "vercelIntegration",
      "keyAuth",
    ]).notNull(),
    resourceId: varchar("resource_id", { length: 256 }).notNull(),
    /**
     * For any additional tags
     */
    tags: json("tags").$type<unknown>(),
  },
  (table) => ({
    resourceIdIdx: index("resource_id_idx").on(table.resourceId),
    actorIdIdx: index("actor_id_idx").on(table.actorId),
  }),
);

export const auditLogsRelations = relations(auditLogs, ({ one, many }) => ({
  key: one(keys, {
    fields: [auditLogs.resourceId],
    references: [keys.id],
  }),
  api: one(apis, {
    fields: [auditLogs.resourceId],
    references: [apis.id],
  }),
  workspace: one(workspaces, {
    fields: [auditLogs.resourceId],
    references: [workspaces.id],
  }),
  changes: many(auditLogChanges),
}));

export const auditLogChanges = mysqlTable(
  "audit_log_changes",
  {
    auditLogId: varchar("audit_log_id", { length: 256 }),
    field: varchar("field", { length: 256 }),
    old: varchar("old", { length: 1024 }),
    new: varchar("new", { length: 1024 }),
  },
  (table) => ({
    primary: primaryKey(table.auditLogId, table.field),
  }),
);

export const auditLogChangesRelations = relations(auditLogChanges, ({ one }) => ({
  auditLog: one(auditLogs, {
    fields: [auditLogChanges.auditLogId],
    references: [auditLogs.id],
  }),
}));
