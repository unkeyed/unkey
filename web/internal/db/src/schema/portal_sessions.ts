import { relations } from "drizzle-orm";
import { bigint, boolean, index, json, mysqlTable } from "drizzle-orm/mysql-core";
import { portalConfigurations } from "./portal_configurations";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
// import { id } from "./util/id";
import { workspaces } from "./workspaces";

export const portalSessions = mysqlTable(
  "portal_sessions",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    // id: id("id").notNull().unique(),
    id: caseSensitiveVarchar("id", { length: 64 }).notNull().unique(),
    // workspaceId: id("workspace_id").notNull(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 256 }).notNull(),
    // portalConfigId: id("portal_config_id").notNull(),
    portalConfigId: caseSensitiveVarchar("portal_config_id", { length: 64 }).notNull(),
    externalId: caseSensitiveVarchar("external_id", { length: 256 }).notNull(),
    permissions: json("permissions").$type<string[]>().notNull(),
    preview: boolean("preview").notNull().default(false),
    expiresAt: bigint("expires_at", { mode: "number" }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
  },
  (table) => [
    index("idx_workspace").on(table.workspaceId),
    index("idx_external_id").on(table.externalId),
    index("idx_expires").on(table.expiresAt),
  ],
);

export const portalSessionsRelations = relations(portalSessions, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [portalSessions.workspaceId],
    references: [workspaces.id],
  }),
  portalConfiguration: one(portalConfigurations, {
    fields: [portalSessions.portalConfigId],
    references: [portalConfigurations.id],
  }),
}));
