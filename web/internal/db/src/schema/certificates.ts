import { relations } from "drizzle-orm";
import { bigint, mysqlTable, text, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const certificates = mysqlTable(
  "certificates",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    hostname: varchar("hostname", { length: 255 }).notNull(),
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
