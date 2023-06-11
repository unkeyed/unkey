import { mysqlTable, varchar, json, datetime } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { apis } from "./apis";
import { tenants } from "./tenants";

export const keys = mysqlTable("keys", {
  id: varchar("id", { length: 256 }).primaryKey(),
  apiId: varchar("api_id", { length: 256 }).notNull(),
  tenantId: varchar("tenant_id", { length: 256 }).notNull(),
  hash: varchar("hash", { length: 256 }).notNull(),
  start: varchar("start", { length: 256 }),
  ownerId: varchar("owner_id", { length: 256 }),
  meta: json("meta"),
  createdAt: datetime("created_at", { fsp: 3 }).notNull(), // unix milli
  expires: datetime("expires", { fsp: 3 }), // unix milli
});

export const keysRelations = relations(keys, ({ one }) => ({
  tenant: one(tenants, {
    fields: [keys.tenantId],
    references: [tenants.id],
  }),
  app: one(apis, {
    fields: [keys.apiId],
    references: [apis.id],
  }),
}));
