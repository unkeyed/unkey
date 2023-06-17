import { mysqlTable, varchar, json, datetime } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { apis } from "./apis";
import { keys } from "./keys";

export const policies = mysqlTable("policies", {
  id: varchar("id", { length: 256 }).primaryKey(),
  name: varchar("name", { length: 256 }),
  apiId: varchar("api_id", { length: 256 }),
  workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
  createdAt: datetime("created_at", { fsp: 3 }).notNull(), // unix milli
  updatedAt: datetime("updated_at", { fsp: 3 }).notNull(), // unix milli
  version: varchar("version", { length: 256, enum: ["v1"] }).notNull(),
  policy: json("policy").notNull(),
});

export const policiesRelations = relations(policies, ({ one, many }) => ({
  api: one(apis, {
    fields: [policies.apiId],
    references: [apis.id],
  }),
  keys: many(keys),
}));
