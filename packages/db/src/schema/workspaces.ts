// db.ts
import { int, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { keys } from "./keys";
import { apis } from "./apis";

export const workspaces = mysqlTable("workspaces", {
  id: varchar("id", { length: 256 }).primaryKey(),
  name: varchar("name", { length: 256 }).notNull(),
  slug: varchar("slug", { length: 256 }).notNull(),
});


export const workspacesRelations = relations(workspaces, ({ many }) => ({
  keys: many(keys),
  apis: many(apis),
}));
