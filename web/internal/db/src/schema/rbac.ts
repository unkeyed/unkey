import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, unique, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { keys } from "./keys";
import { caseInsensitiveVarchar } from "./util/case_insensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const permissions = mysqlTable(
  "permissions",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    name: caseInsensitiveVarchar("name", { length: 512 }).notNull(),
    slug: varchar("slug", { length: 128 }).notNull(),
    description: varchar("description", { length: 512 }),
    createdAtM: bigint("created_at_m", { mode: "number" })
      .notNull()
      .default(0)
      .$defaultFn(() => Date.now()),
    updatedAtM: bigint("updated_at_m", { mode: "number" }).$onUpdateFn(() => Date.now()),
  },
  (table) => [
    unique("unique_slug_per_workspace_idx").on(table.workspaceId, table.slug),
    index("permissions_project_id_idx").on(table.projectId),
  ],
);
export const permissionsRelations = relations(permissions, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [permissions.workspaceId],
    references: [workspaces.id],
  }),
  keys: many(keysPermissions, {
    relationName: "permissions_keys_permissions_relations",
  }),
  roles: many(rolesPermissions, {
    relationName: "roles_permissions",
  }),
}));

export const keysPermissions = mysqlTable(
  "keys_permissions",
  {
    pk: primaryKey(),
    keyId: id("key_id").notNull(),
    permissionId: id("permission_id").notNull(),
    workspaceId: id("workspace_id").notNull(),

    createdAtM: bigint("created_at_m", { mode: "number" })
      .notNull()
      .default(0)
      .$defaultFn(() => Date.now()),
    updatedAtM: bigint("updated_at_m", { mode: "number" }).$onUpdateFn(() => Date.now()),
  },
  (table) => [
    unique("keys_permissions_key_id_permission_id_workspace_id").on(
      table.keyId,
      table.permissionId,
      table.workspaceId,
    ),
    unique("key_id_permission_id_idx").on(table.keyId, table.permissionId),
  ],
);

export const keysPermissionsRelations = relations(keysPermissions, ({ one }) => ({
  key: one(keys, {
    fields: [keysPermissions.keyId],
    references: [keys.id],
    relationName: "keys_keys_permissions_relations",
  }),
  permission: one(permissions, {
    fields: [keysPermissions.permissionId],
    references: [permissions.id],
    relationName: "permissions_keys_permissions_relations",
  }),
}));
export const roles = mysqlTable(
  "roles",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    name: caseInsensitiveVarchar("name", { length: 512 }).notNull(),
    description: varchar("description", { length: 512 }),
    createdAtM: bigint("created_at_m", { mode: "number" })
      .notNull()
      .default(0)
      .$defaultFn(() => Date.now()),
    updatedAtM: bigint("updated_at_m", { mode: "number" }).$onUpdateFn(() => Date.now()),
  },
  (table) => [
    index("workspace_id_idx").on(table.workspaceId),
    unique("unique_name_per_workspace_idx").on(table.name, table.workspaceId),
    index("roles_project_id_idx").on(table.projectId),
  ],
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

export const rolesPermissions = mysqlTable(
  "roles_permissions",
  {
    pk: primaryKey(),
    roleId: id("role_id").notNull(),
    permissionId: id("permission_id").notNull(),
    workspaceId: id("workspace_id").notNull(),

    createdAtM: bigint("created_at_m", { mode: "number" })
      .notNull()
      .default(0)
      .$defaultFn(() => Date.now()),
    updatedAtM: bigint("updated_at_m", { mode: "number" }).$onUpdateFn(() => Date.now()),
  },
  (table) => [
    unique("roles_permissions_role_id_permission_id_workspace_id").on(
      table.roleId,
      table.permissionId,
      table.workspaceId,
    ),
    unique("unique_tuple_permission_id_role_id").on(table.permissionId, table.roleId),
  ],
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
    pk: primaryKey(),
    keyId: id("key_id").notNull(),
    roleId: id("role_id").notNull(),
    workspaceId: id("workspace_id").notNull(),

    createdAtM: bigint("created_at_m", { mode: "number" })
      .notNull()
      .default(0)
      .$defaultFn(() => Date.now()),
    updatedAtM: bigint("updated_at_m", { mode: "number" }).$onUpdateFn(() => Date.now()),
  },
  (table) => [
    unique("keys_roles_role_id_key_id_workspace_id").on(
      table.roleId,
      table.keyId,
      table.workspaceId,
    ),
    uniqueIndex("unique_key_id_role_id").on(table.keyId, table.roleId),
  ],
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
