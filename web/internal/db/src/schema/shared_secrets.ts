import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
// import { id } from "./util/id";
import { embeddedEncrypted } from "./util/embedded_encrypted";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

// One-time links for sharing a secret. Holds only the vault ciphertext (keyring
// = workspace id), like `encrypted_keys`. Single-use: revealing deletes the row.
// `expires_at` time-boxes it; there is no native TTL, so cleanup is lazy.
export const sharedSecrets = mysqlTable(
  "shared_secrets",
  {
    pk: primaryKey(),
    // id: id("id").notNull().unique(),
    id: caseSensitiveVarchar("id", { length: 256 }).notNull().unique(),
    // workspaceId: id("workspace_id").notNull(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 256 }).notNull(),
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
