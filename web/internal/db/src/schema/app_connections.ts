import { index, mysqlEnum, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";

// A connection lets deployments of one app environment (the source, X) reach
// another app (the target, Y) over private networking. Each row is an editable
// default: deployment creation copies it into deploymentConnections, so edits
// only affect future deployments of X.
//
// This table holds what every connection needs: the source, the target, and the
// name. How to choose a deployment of an app target lives in
// connectionAppTargets, so this table carries no app-only selection columns.
export const appConnections = mysqlTable(
  "app_connections",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    // The source app environment whose deployments use this connection.
    appId: id("app_id").notNull(),
    environmentId: id("environment_id").notNull(),
    // The target. Apps are the only supported resource, so resource_id is an
    // app id.
    resourceType: mysqlEnum("resource_type", ["app"]).notNull(),
    resourceId: id("resource_id").notNull(),
    // How source deployments address the target: a connection named database
    // resolves as database.unkey.internal, injected as DATABASE_HOST. Unique
    // per source app environment.
    name: caseSensitiveVarchar("name", { length: 63 }).notNull(),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("app_connections_app_name_idx").on(table.appId, table.environmentId, table.name),
    uniqueIndex("app_connections_app_resource_idx").on(
      table.appId,
      table.environmentId,
      table.resourceType,
      table.resourceId,
    ),
    index("app_connections_project_idx").on(table.projectId),
    index("app_connections_resource_idx").on(table.resourceType, table.resourceId),
    index("app_connections_workspace_idx").on(table.workspaceId, table.resourceType),
  ],
);
