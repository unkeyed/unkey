import { relations } from "drizzle-orm";
import { bigint, boolean, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { portalConfigurations } from "./portal_configurations";
import { workspaces } from "./workspaces";

export const portalSessionTokens = mysqlTable(
  "portal_session_tokens",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    portalConfigId: varchar("portal_config_id", { length: 64 }).notNull(),
    externalId: varchar("external_id", { length: 256 }).notNull(),
    metadata: json("metadata").$type<Record<string, unknown>>(),
    permissions: json("permissions").$type<string[]>().notNull(),
    preview: boolean("preview").notNull().default(false),
    exchangedAt: bigint("exchanged_at", { mode: "number" }),
    expiresAt: bigint("expires_at", { mode: "number" }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
  },
  (table) => [
    index("idx_workspace").on(table.workspaceId),
    index("idx_expires").on(table.expiresAt),
  ],
);

export const portalSessionTokensRelations = relations(portalSessionTokens, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [portalSessionTokens.workspaceId],
    references: [workspaces.id],
  }),
  portalConfiguration: one(portalConfigurations, {
    fields: [portalSessionTokens.portalConfigId],
    references: [portalConfigurations.id],
  }),
}));
