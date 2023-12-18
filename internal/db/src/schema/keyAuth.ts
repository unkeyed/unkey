import { relations } from "drizzle-orm";
import { datetime, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { keys } from "./keys";
import { workspaces } from "./workspaces";

export const keyAuth = mysqlTable("key_auth", {
  id: varchar("id", { length: 256 }).primaryKey(),
  workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
  createdAt: datetime("created_at", { fsp: 3 }),
  deletedAt: datetime("deleted_at", { fsp: 3 }),
});

export const keyAuthRelations = relations(keyAuth, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [keyAuth.workspaceId],
    references: [workspaces.id],
  }),
  api: one(apis, {
    fields: [keyAuth.id],
    references: [apis.keyAuthId],
  }),
  keys: many(keys),
}));
