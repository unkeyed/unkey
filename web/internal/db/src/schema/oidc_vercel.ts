import { relations } from "drizzle-orm";
import { boolean, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { permissions } from "./rbac";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const oidcVercel = mysqlTable(
  "oidc_vercel",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    name: varchar("name", { length: 256 }).notNull(),
    audience: caseSensitiveVarchar("audience", { length: 512 }).notNull().unique(),
    vercelTeamSlug: caseSensitiveVarchar("vercel_team_slug", { length: 100 }).notNull(),
    vercelProjectName: caseSensitiveVarchar("vercel_project_name", { length: 100 }).notNull(),
    vercelEnvironment: caseSensitiveVarchar("vercel_environment", { length: 100 }).notNull(),
    enabled: boolean("enabled").notNull().default(true),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("workspace_vercel_team_project_environment_idx").on(
      table.workspaceId,
      table.vercelTeamSlug,
      table.vercelProjectName,
      table.vercelEnvironment,
    ),
    index("workspace_id_idx").on(table.workspaceId),
  ],
);

export const oidcVercelRelations = relations(oidcVercel, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [oidcVercel.workspaceId],
    references: [workspaces.id],
  }),
  permissions: many(oidcVercelPermissions),
}));

export const oidcVercelPermissions = mysqlTable(
  "oidc_vercel_permissions",
  {
    pk: primaryKey(),
    oidcVercelId: id("oidc_vercel_id").notNull(),
    permissionId: id("permission_id").notNull(),
    workspaceId: id("workspace_id").notNull(),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("oidc_vercel_permission_idx").on(table.oidcVercelId, table.permissionId),
    index("permission_id_idx").on(table.permissionId),
  ],
);

export const oidcVercelPermissionsRelations = relations(oidcVercelPermissions, ({ one }) => ({
  oidcVercel: one(oidcVercel, {
    fields: [oidcVercelPermissions.oidcVercelId],
    references: [oidcVercel.id],
  }),
  permission: one(permissions, {
    fields: [oidcVercelPermissions.permissionId],
    references: [permissions.id],
  }),
  workspace: one(workspaces, {
    fields: [oidcVercelPermissions.workspaceId],
    references: [workspaces.id],
  }),
}));
