import { relations, sql } from "drizzle-orm";
import { bigint, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

// Provider connections established via OAuth. One grant can back multiple log_drains
// (and any future feature that needs a provider connection), so customers connect
// e.g. Datadog once and reuse it across every project.
export const oauthGrants = mysqlTable(
  "oauth_grants",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    // Lowercase provider key. e.g. 'datadog'. 'axiom' is reserved for when Axiom exposes
    // an OAuth flow for ingest tokens.
    provider: varchar("provider", { length: 64 }).notNull(),

    // Human-readable label shown in the dashboard, e.g. "Datadog org acme.datadoghq.com".
    accountLabel: varchar("account_label", { length: 256 }).notNull(),

    // Provider-specific region, e.g. Datadog 'us', 'eu', 'us3', 'us5', 'ap1'.
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
