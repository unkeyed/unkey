import { relations, sql } from "drizzle-orm";
import { bigint, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

// Provider connections established via OAuth. One grant can back multiple log_drains
// (and any future feature that needs a provider connection).
export const oauthGrants = mysqlTable(
  "oauth_grants",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    // Lowercase provider key.
    provider: varchar("provider", { length: 64 }).notNull(),

    // Human-readable label shown in the dashboard.
    accountLabel: varchar("account_label", { length: 256 }).notNull(),

    // Provider-specific region identifier.
    region: varchar("region", { length: 32 }),

    scopes: json("scopes").$type<string[]>().notNull().default(sql`('[]')`),

    // Vault-encrypted access token (and refresh token where applicable, joined with a
    // separator the application layer knows about).
    encryptedCredentials: varchar("encrypted_credentials", { length: 1024 }).notNull(),
    encryptionKeyId: varchar("encryption_key_id", { length: 256 }).notNull(),

    expiresAt: bigint("expires_at", { mode: "number" }),
    revokedAt: bigint("revoked_at", { mode: "number" }),

    ...lifecycleDates,
  },
  (table) => [
    index("oauth_grants_workspace_idx").on(table.workspaceId, table.provider, table.revokedAt),
  ],
);

export const oauthGrantsRelations = relations(oauthGrants, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [oauthGrants.workspaceId],
    references: [workspaces.id],
  }),
}));
