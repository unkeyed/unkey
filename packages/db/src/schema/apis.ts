import { mysqlTable, json, varchar, index } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { workspaces } from "./workspaces";
import { keys } from "./keys";

export const apis = mysqlTable("apis", {
  id: varchar("id", { length: 256 }).primaryKey(),
  name: varchar("name", { length: 256 }).notNull(),
  workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
});

export const apisRelations = relations(apis, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [apis.workspaceId],
    references: [workspaces.id],
  }),
  keys: many(keys),
}));
