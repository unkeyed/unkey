import { relations } from "drizzle-orm";
import { index, mysqlTable, primaryKey, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { keys } from "./keys";
import { workspaces } from "./workspaces";

export const permissions = mysqlTable(
  "permissions",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    name: varchar("name", { length: 512 }).notNull(),
    key: varchar("key", { length: 512 }).notNull(),
    description: varchar("description", { length: 512 }),
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
    relationName: "permissions_relations",
  }),
}));

export const keysPermissions = mysqlTable(
  "keys_permissions",
  {
    keyId: varchar("key_id", { length: 256 }).notNull(),
    permissionId: varchar("permission_id", { length: 256 })
      .notNull()
      .references(() => permissions.id, { onDelete: "cascade" }),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    // .references(() => workspaces.id, { onDelete: "cascade" }),
  },
  (table) => ({
    pk: primaryKey({ columns: [table.keyId, table.permissionId] }),
  }),
);

export const keysPermissionsRelations = relations(keysPermissions, ({ one }) => ({
  key: one(keys, {
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

export const roles = mysqlTable(
  "roles",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 })
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),
    name: varchar("name", { length: 512 }).notNull(),
    description: varchar("description", { length: 512 }),
    key: varchar("key", { length: 512 }).notNull(),
  },
  (table) => ({
    uniqueNamePerWorkspace: uniqueIndex("unique_name_per_workspace_idx").on(
      table.name,
      table.workspaceId,
    ),
    uniqueKeyPerWorkspace: uniqueIndex("unique_key_per_workspace_idx").on(
      table.key,
      table.workspaceId,
    ),
    workspaceIdIndex: index("workspace_id_idx").on(table.workspaceId),
  }),
);
export const rolesRelations = relations(roles, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [roles.workspaceId],
    references: [workspaces.id],
  }),
  keys: many(keysRoles, {
    relationName: "keys_roles_roles_relations",
  }),
  permissions: many(rolesPermissions, {
    relationName: "roles_rolesPermissions",
  }),
}));

/**
 * N:M table to connect roles and permissions
 */
export const rolesPermissions = mysqlTable(
  "roles_permissions",
  {
    roleId: varchar("role_id", { length: 256 })
      .notNull()
      .references(() => roles.id, { onDelete: "cascade" }),
    permissionId: varchar("permission_id", { length: 256 })
      .notNull()
      .references(() => permissions.id, { onDelete: "cascade" }),
    workspaceId: varchar("workspace_id", { length: 256 })
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),
  },
  (table) => ({
    primaryKey: primaryKey({ columns: [table.roleId, table.permissionId, table.workspaceId] }),
    uniquePermissionIdRoleId: uniqueIndex("unique_tuple_permission_id_role_id").on(
      table.permissionId,
      table.roleId,
    ),
  }),
);

export const rolesPermissionsRelations = relations(rolesPermissions, ({ one }) => ({
  role: one(roles, {
    fields: [rolesPermissions.roleId],
    references: [roles.id],
    relationName: "roles_rolesPermissions",
  }),
  permission: one(permissions, {
    fields: [rolesPermissions.permissionId],
    references: [permissions.id],
    relationName: "roles_permissions",
  }),
}));

export const keysRoles = mysqlTable(
  "keys_roles",
  {
    keyId: varchar("key_id", { length: 256 }).notNull(),
    roleId: varchar("role_id", { length: 256 })
      .notNull()
      .references(() => roles.id, { onDelete: "cascade" }),
    workspaceId: varchar("workspace_id", { length: 256 })
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),
  },
  (table) => ({
    primaryKey: primaryKey({ columns: [table.roleId, table.keyId, table.workspaceId] }),

    uniqueTuples: uniqueIndex("unique_key_id_role_id").on(table.keyId, table.roleId),
  }),
);

export const keysRolesRelations = relations(keysRoles, ({ one }) => ({
  role: one(roles, {
    fields: [keysRoles.roleId],
    references: [roles.id],
    relationName: "keys_roles_roles_relations",
  }),
  key: one(keys, {
    fields: [keysRoles.keyId],
    references: [keys.id],
    relationName: "keys_roles_key_relations",
  }),
}));
