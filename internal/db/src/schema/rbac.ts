import { relations } from "drizzle-orm";
import { index, mysqlTable, primaryKey, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
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

export const permissions = mysqlTable(
  "permissions",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    name: varchar("name", { length: 512 }).notNull(),
  },
  (table) => ({
    uniqueNamePerWorkspace: uniqueIndex("unique_name_per_workspace_idx").on(
      table.name,
      table.workspaceId,
    ),
    workspaceIdIndex: index("workspace_id_idx").on(table.workspaceId),
  }),
);

export const permissionsRelations = relations(permissions, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [permissions.workspaceId],
    references: [workspaces.id],
  }),
  keys: many(keysPermissions, {
    relationName: "keys_permissions_relation",
  }),
}));

export const keysPermissions = mysqlTable(
  "keys_permissions",
  {
    keyId: varchar("key_id", { length: 256 }).notNull(),
    permissionId: varchar("permission_id", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
  },
  (table) => ({
    pk: primaryKey({ columns: [table.keyId, table.permissionId] }),
  }),
);

export const keysPermissionsRelations = relations(keysPermissions, ({ one }) => ({
  keys: one(keys, {
    fields: [keysPermissions.keyId],
    references: [keys.id],
    relationName: "keys_permissions_relations",
  }),
  permission: one(permissions, {
    fields: [keysPermissions.permissionId],
    references: [permissions.id],
    relationName: "permissions_relations",
  }),
}));
