import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { embeddedEncrypted } from "./util/embedded_encrypted";
import { workspaces } from "./workspaces";

// One-time links for sharing a secret. Holds only the vault ciphertext (keyring
// = workspace id), like `encrypted_keys`. Single-use: revealing deletes the row.
// `expires_at` time-boxes it; there is no native TTL, so cleanup is lazy.
export const sharedSecrets = mysqlTable(
  "shared_secrets",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    expiresAt: bigint("expires_at", { mode: "number" }).notNull(),
    // Rows are write-once (created, then deleted), so there is no updated_at.
    createdAt: bigint("created_at", { mode: "number" })
      .notNull()
      .$defaultFn(() => Date.now()),

    ...embeddedEncrypted,
  },
  // expires_at is indexed for the lazy cleanup sweep. workspace_id is stored for
  // the vault keyring and audit, but nothing queries by it, so it is not indexed.
  (table) => [index("expires_at_idx").on(table.expiresAt)],
);

export const sharedSecretsRelations = relations(sharedSecrets, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [sharedSecrets.workspaceId],
    references: [workspaces.id],
  }),
}));
