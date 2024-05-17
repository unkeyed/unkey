import { relations } from "drizzle-orm";
import { int, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { embeddedSecret } from "./util/embedded_secret";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const dataEncryptionKeys = mysqlTable(
  "data_encryption_keys",
  {
    id: varchar("id", { length: 256 }).primaryKey(),

    version: int("version").notNull(),
    workspaceId: varchar("workspace_id", { length: 256 })
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),
    ...embeddedSecret,
    ...lifecycleDates,
  },
  (table) => ({
    oneVersionPerWorkspace: uniqueIndex("workspace_id_version_idx").on(
      table.workspaceId,
      table.version,
    ),
  }),
);

export const dataEncryptionKeysRelations = relations(dataEncryptionKeys, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [dataEncryptionKeys.workspaceId],
    references: [workspaces.id],
  }),
}));
