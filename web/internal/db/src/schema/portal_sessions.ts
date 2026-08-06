import { relations } from "drizzle-orm";
import { bigint, index, json, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { portals } from "./portals";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const portalSessions = mysqlTable(
  "portal_sessions",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    portalId: id("portal_id").notNull(),
    externalId: caseSensitiveVarchar("external_id", { length: 256 }).notNull(),
    permissions: json("permissions").$type<string[]>().notNull(),
    exchangeCodeHash: caseSensitiveVarchar("exchange_code_hash", { length: 44 }),
    exchangeCodeExpiresAt: bigint("exchange_code_expires_at", {
      mode: "number",
    }).notNull(),
    accessTokenHash: caseSensitiveVarchar("access_token_hash", { length: 44 }),
    accessTokenCreatedAt: bigint("access_token_created_at", { mode: "number" }),
    accessTokenExpiresAt: bigint("access_token_expires_at", { mode: "number" }),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
  },
  (table) => [
    uniqueIndex("exchange_code_hash_idx").on(table.exchangeCodeHash),
    uniqueIndex("access_token_hash_idx").on(table.accessTokenHash),
    index("idx_workspace").on(table.workspaceId),
    index("idx_external_id").on(table.externalId),
    index("idx_exchange_code_expires").on(table.exchangeCodeExpiresAt),
    index("idx_access_token_expires").on(table.accessTokenExpiresAt),
  ],
);

export const portalSessionsRelations = relations(portalSessions, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [portalSessions.workspaceId],
    references: [workspaces.id],
  }),
  portal: one(portals, {
    fields: [portalSessions.portalId],
    references: [portals.id],
  }),
}));
