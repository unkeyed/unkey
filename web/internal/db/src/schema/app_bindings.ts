import { index, mysqlEnum, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";

export const appBindings = mysqlTable(
  "app_bindings",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    appId: id("app_id").notNull(),
    environmentId: id("environment_id").notNull(),
    resourceType: caseSensitiveVarchar("resource_type", { length: 32 }).notNull(),
    resourceId: id("resource_id").notNull(),
    name: caseSensitiveVarchar("name", { length: 63 }).notNull(),
    selectionMode: mysqlEnum("selection_mode", ["automatic", "environment", "deployment"]),
    targetEnvironmentId: id("target_environment_id"),
    targetDeploymentId: id("target_deployment_id"),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("app_bindings_app_name_idx").on(table.appId, table.environmentId, table.name),
    uniqueIndex("app_bindings_app_resource_idx").on(
      table.appId,
      table.environmentId,
      table.resourceType,
      table.resourceId,
    ),
    index("app_bindings_project_idx").on(table.projectId),
    index("app_bindings_resource_idx").on(table.resourceType, table.resourceId),
  ],
);
