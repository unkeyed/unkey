import { relations } from "drizzle-orm";
import { bigint, mysqlTable, text, uniqueIndex } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { workspaces } from "./workspaces";

export const certificates = mysqlTable(
  "certificates",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: caseSensitiveVarchar("id", { length: 64 }).notNull().unique(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 255 }).notNull(),
    hostname: caseSensitiveVarchar("hostname", { length: 255 }).notNull(),
    certificate: text("certificate").notNull(),
    encryptedPrivateKey: text("encrypted_private_key").notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => [uniqueIndex("unique_hostname").on(table.hostname)],
);

export const certificatesRelations = relations(certificates, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [certificates.workspaceId],
    references: [workspaces.id],
  }),
}));
