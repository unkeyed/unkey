import { relations } from "drizzle-orm";
import { mysqlTable, primaryKey, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { keys } from "./keys";
import { workspaces } from "./workspaces";

export const roles = mysqlTable(
  "roles",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 })
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),
    apiId: varchar("api_id", { length: 256 })
      .notNull()
      .references(() => apis.id, { onDelete: "cascade" }),
    name: varchar("name", { length: 512 }).notNull(),
  },
  (table) => ({
    uniqueNamePerApi: uniqueIndex("unique_name_per_api").on(table.name, table.apiId),
  }),
);

export const rolesRelations = relations(roles, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [roles.workspaceId],
    references: [workspaces.id],
  }),
  api: one(apis, {
    fields: [roles.id],
    references: [apis.keyAuthId],
  }),
  rolesToKeys: many(rolesToKeys),
}));

export const rolesToKeys = mysqlTable(
  "roles_to_keys",
  {
    keyId: varchar("key_id", { length: 256 })
      .notNull()
      .references(() => keys.id, { onDelete: "cascade" }),
    roleId: varchar("role_id", { length: 256 })
      .notNull()
      .references(() => roles.id, { onDelete: "cascade" }),
  },
  (table) => ({
    pk: primaryKey(table.keyId, table.roleId),
  }),
);

export const rolesToKeysRelations = relations(rolesToKeys, ({ one }) => ({
  key: one(keys, {
    fields: [rolesToKeys.keyId],
    references: [keys.id],
  }),
  role: one(roles, {
    fields: [rolesToKeys.roleId],
    references: [roles.id],
  }),
}));
