// db.ts
import { int, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { keys } from "./keys";

export const tenants = mysqlTable("tenants", {
  id: varchar("id", { length: 256 }).primaryKey(),
  name: varchar("name", { length: 256 }).notNull(),
  slug: varchar("slug", { length: 256 }).notNull(),
});

export const tenantsRelations = relations(tenants, ({ many }) => ({
  keys: many(keys),
}));
