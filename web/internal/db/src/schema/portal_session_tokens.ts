import { relations } from "drizzle-orm";
import { bigint, boolean, index, json, mysqlTable } from "drizzle-orm/mysql-core";
import { portalConfigurations } from "./portal_configurations";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const portalSessionTokens = mysqlTable(
  "portal_session_tokens",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    portalConfigId: id("portal_config_id").notNull(),
    externalId: caseSensitiveVarchar("external_id", { length: 256 }).notNull(),
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
