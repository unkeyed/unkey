import { mysqlTable, json, varchar } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { tenants } from "./tenants";
import { keys } from "./keys";

export const apis = mysqlTable("apis", {
  id: varchar("id", { length: 256 }).primaryKey(),
  name: varchar("name", { length: 256 }).notNull(),
  tenantId: varchar("tenant_id", { length: 256 }).notNull(),
});

export const apisRelations = relations(apis, ({ one, many }) => ({
  tenant: one(tenants, {
    fields: [apis.tenantId],
    references: [tenants.id],
  }),
  keys: many(keys),
}));
