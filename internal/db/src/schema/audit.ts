import { relations } from "drizzle-orm";
import { datetime, json, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { keys } from "./keys";
import { workspaces } from "./workspaces";

export const auditLogs = mysqlTable("audit_logs", {
  id: varchar("id", { length: 256 }).primaryKey(),

  /**
   * A machine readable description of what happened
   */
  event: mysqlEnum("event", [
    "workspace.create",
    "workspace.update",
    "workspace.delete",
    "api.create",
    "api.update",
    "api.delete",
    "key.create",
    "key.update",
    "key.delete",
    "vercelIntegration.create",
    "vercelIntegration.update",
    "vercelIntegration.delete",
    "vercelBinding.create",
    "vercelBinding.update",
    "vercelBinding.delete",
  ]).notNull(),

  /**
   * A human readable description of what happened.
   */
  description: varchar("description", { length: 512 }).notNull(),
  time: datetime("time", { fsp: 3 }).notNull(), // unix milli

  actorType: mysqlEnum("actor_type", ["user", "key"]).notNull(),
  actorId: varchar("actor_id", { length: 256 }).notNull(),

  workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
  apiId: varchar("api_id", { length: 256 }),
  keyId: varchar("key_id", { length: 256 }),
  keyAuthId: varchar("key_auth_id", { length: 256 }),
  vercelIntegrationId: varchar("vercel_integration_id", { length: 256 }),
  vercelBindingId: varchar("vercel_binding_id", { length: 256 }),

  /**
   * For any additional tags
   */
  tags: json("tags").$type<unknown>(),
});

export const auditLogsRelations = relations(auditLogs, ({ one }) => ({
  key: one(keys, {
    fields: [auditLogs.keyId],
    references: [keys.id],
  }),
  api: one(apis, {
    fields: [auditLogs.apiId],
    references: [apis.id],
  }),
  workspace: one(workspaces, {
    fields: [auditLogs.workspaceId],
    references: [workspaces.id],
  }),
}));
