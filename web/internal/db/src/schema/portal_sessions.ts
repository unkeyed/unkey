import { relations } from "drizzle-orm";
import { bigint, boolean, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { portalConfigurations } from "./portal_configurations";
import { workspaces } from "./workspaces";

export const portalSessions = mysqlTable(
  "portal_sessions",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    portalConfigId: varchar("portal_config_id", { length: 64 }).notNull(),
    externalId: varchar("external_id", { length: 256 }).notNull(),
    metadata: json("metadata").$type<Record<string, unknown>>(),
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
