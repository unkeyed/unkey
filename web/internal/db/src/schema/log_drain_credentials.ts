import { relations } from "drizzle-orm";
import { bigint, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { logDrains } from "./log_drains";
import { oauthGrants } from "./oauth_grants";

// Credentials live in their own table so token rotation does not churn the config row
// or its binlog entries. One row per drain.
export const logDrainCredentials = mysqlTable("log_drain_credentials", {
  drainId: varchar("drain_id", { length: 64 }).notNull().primaryKey(),

  // 'paste' means the customer pasted a token (encrypted via Vault).
  // 'oauth' means the drain references an oauth_grants row.
  source: mysqlEnum("source", ["paste", "oauth"]).notNull(),

  // Populated when source = 'paste'. Base64-encoded ciphertext from Vault.
  encryptedCredentials: varchar("encrypted_credentials", { length: 1024 }),
  encryptionKeyId: varchar("encryption_key_id", { length: 256 }),

  // Populated when source = 'oauth'. References oauth_grants.id.
  oauthGrantId: varchar("oauth_grant_id", { length: 64 }),

  updatedAt: bigint("updated_at", { mode: "number" })
    .notNull()
    .$onUpdateFn(() => Date.now()),
});

export const logDrainCredentialsRelations = relations(logDrainCredentials, ({ one }) => ({
  drain: one(logDrains, {
    fields: [logDrainCredentials.drainId],
    references: [logDrains.id],
  }),
  oauthGrant: one(oauthGrants, {
    fields: [logDrainCredentials.oauthGrantId],
    references: [oauthGrants.id],
  }),
}));
