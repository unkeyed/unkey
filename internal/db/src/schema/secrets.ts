import { relations } from "drizzle-orm";
import { mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { embeddedEncrypted } from "./util/embedded_encrypted";
import { workspaces } from "./workspaces";

export const secrets = mysqlTable("secrets", {
  id: varchar("id", { length: 256 }).primaryKey(),
  workspaceId: varchar("workspace_id", { length: 256 })
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),
  ...embeddedEncrypted,
});

export const secretsRelations = relations(secrets, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [secrets.workspaceId],
    references: [workspaces.id],
  }),
}));
