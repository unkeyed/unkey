import { relations } from "drizzle-orm";
// db.ts
import { datetime, mysqlEnum, mysqlTable, primaryKey, varchar } from "drizzle-orm/mysql-core";
import { keys } from "./keys";
import { workspaces } from "./workspaces";

export const vercelIntegrations = mysqlTable("vercel_integrations", {
  id: varchar("id", { length: 256 }).primaryKey(),
  workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
  vercelTeamId: varchar("team_id", { length: 256 }),
});

export const vercelBindings = mysqlTable(
  "vercel_bindings",
  {
    integrationId: varchar("integration_id", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    environment: mysqlEnum("environment", ["development", "preview", "production"]).notNull(),
    apiId: varchar("api_id", { length: 256 }).notNull(),
    rootKeyId: varchar("root_key_id", { length: 256 }).notNull(),
    createdAt: datetime("created_at", { fsp: 3 }).notNull(),
    updatedAt: datetime("updated_at", { fsp: 3 }).notNull(),
    // userId
    lastEditedBy: varchar("last_edited_by", { length: 256 }).notNull(),
  },
  (table) => ({
    pk: primaryKey(table.projectId, table.environment),
  }),
);

export const vercelIntegrationRelations = relations(vercelIntegrations, ({ many, one }) => ({
  workspace: one(workspaces, {
    fields: [vercelIntegrations.workspaceId],
    references: [workspaces.id],
  }),
  keys: many(keys),
  vercelBindings: many(vercelBindings),
}));

export const vercelBindingRelations = relations(vercelBindings, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [vercelBindings.workspaceId],
    references: [workspaces.id],
  }),
  vercelIntegrations: one(vercelIntegrations, {
    fields: [vercelBindings.integrationId],
    references: [vercelIntegrations.id],
  }),
}));
