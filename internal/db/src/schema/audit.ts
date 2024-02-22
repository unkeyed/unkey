import { relations } from "drizzle-orm";
import { datetime, json, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { keys } from "./keys";
import { workspaces } from "./workspaces";

export const auditLogs = mysqlTable("audit_logs", {
  id: varchar("id", { length: 256 }).primaryKey(),
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
    "role.create",
    "role.update",
    "role.delete",
    "permission.create",
    "permission.update",
    "permission.delete",
    "authorization.connect_role_and_permission",
    "authorization.disconnect_role_and_permissions",
    "authorization.connect_role_and_key",
    "authorization.disconnect_role_and_key",
    "authorization.connect_permission_and_key",
    "authorization.disconnect_permission_and_key",
  ]).notNull(),
  description: varchar("description", { length: 512 }).notNull(),
  time: datetime("time", { mode: "date", fsp: 3 }).notNull(),
  actorType: mysqlEnum("actor_type", ["user", "key"]).notNull(),
  actorId: varchar("actor_id", { length: 256 }).notNull(),
  workspaceId: varchar("workspace_id", { length: 256 })
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),
  apiId: varchar("api_id", { length: 256 }),
  keyId: varchar("key_id", { length: 256 }),
  keyAuthId: varchar("key_auth_id", { length: 256 }),
  vercelIntegrationId: varchar("vercel_integration_id", { length: 256 }),
  vercelBindingId: varchar("vercel_binding_id", { length: 256 }),
  roleId: varchar("role_id", { length: 256 }),
  permissionId: varchar("permission_id", { length: 256 }),
  tags: json("tags").$type<Record<string, string | number | boolean>>(),
  ipAddress: varchar("ip_address", { length: 256 }),
  userAgent: varchar("user_agent", { length: 256 }),
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
