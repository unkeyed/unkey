import { relations } from "drizzle-orm";
// db.ts
import { bigint, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const vercelIntegrations = mysqlTable("vercel_integrations", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  id: varchar("id", { length: 256 }).notNull().unique(),
  workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
  vercelTeamId: varchar("team_id", { length: 256 }),
  accessToken: varchar("access_token", { length: 256 }).notNull(),
  ...lifecycleDatesMigration,
});

export const vercelBindings = mysqlTable(
  "vercel_bindings",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    integrationId: varchar("integration_id", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    environment: mysqlEnum("environment", ["development", "preview", "production"]).notNull(),
    resourceId: varchar("resource_id", { length: 256 }).notNull(),
    resourceType: mysqlEnum("resource_type", ["rootKey", "apiId"]).notNull(),
    vercelEnvId: varchar("vercel_env_id", { length: 256 }).notNull(),

    //  // userId
    lastEditedBy: varchar("last_edited_by", { length: 256 }).notNull(),
    ...lifecycleDatesMigration,
  },
  (table) => ({
    uniqueProjectEnvironmentResourceIndex: uniqueIndex("project_environment_resource_type_idx").on(
      table.projectId,
      table.environment,
      table.resourceType,
    ),
  }),
);

export const vercelIntegrationRelations = relations(vercelIntegrations, ({ many, one }) => ({
  workspace: one(workspaces, {
    relationName: "vercel_workspace_relation",
    fields: [vercelIntegrations.workspaceId],
    references: [workspaces.id],
  }),
  // keys: many(keys,),
  vercelBindings: many(vercelBindings),
}));

export const vercelBindingRelations = relations(vercelBindings, ({ one }) => ({
  workspace: one(workspaces, {
    relationName: "vercel_key_binding_relation",
    fields: [vercelBindings.workspaceId],
    references: [workspaces.id],
  }),
  vercelIntegrations: one(vercelIntegrations, {
    fields: [vercelBindings.integrationId],
    references: [vercelIntegrations.id],
  }),
}));
