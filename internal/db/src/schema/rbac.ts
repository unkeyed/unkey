import { relations } from "drizzle-orm";
import { index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { keys } from "./keys";
import { workspaces } from "./workspaces";

export const roles = mysqlTable(
  "roles",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    keyId: varchar("key_id", { length: 256 }).notNull(),
    role: varchar("role", { length: 512 }).notNull(),
  },
  (table) => ({
    rolesIndex: index("roles_idx").on(table.role),
    workspaceIdIndex: index("workspace_id_idx").on(table.workspaceId),
    keyIdIndex: index("key_id_idx").on(table.keyId),
    uniqueKeyRoleIndex: uniqueIndex("key_role_idx").on(table.keyId, table.role),
  }),
);

export const rolesRelations = relations(roles, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [roles.workspaceId],
    references: [workspaces.id],
  }),
  key: one(keys, {
    relationName: "key_roles_relation",
    fields: [roles.keyId],
    references: [keys.id],
  }),
}));
