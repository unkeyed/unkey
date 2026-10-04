import { bigint, index, mysqlEnum, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

// The appConnections rows a source deployment was created with, copied at
// creation and never changed. Columns mean the same as in appConnections, with
// deployment_id naming the source deployment. Together with
// deploymentConnectionAppTargets it preserves the names and rules chosen at
// creation: the deployment's host variables came from these names, so it keeps
// them after the defaults are renamed, retargeted, or removed.
export const deploymentConnections = mysqlTable(
  "deployment_connections",
  {
    pk: primaryKey(),
    deploymentId: id("deployment_id").notNull(),
    connectionId: id("connection_id").notNull(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    appId: id("app_id").notNull(),
    environmentId: id("environment_id").notNull(),
    resourceType: mysqlEnum("resource_type", ["app"]).notNull(),
    resourceId: id("resource_id").notNull(),
    name: caseSensitiveVarchar("name", { length: 63 }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
  },
  (table) => [
    uniqueIndex("deployment_connections_deployment_connection_idx").on(
      table.deploymentId,
      table.connectionId,
    ),
    uniqueIndex("deployment_connections_deployment_name_idx").on(table.deploymentId, table.name),
    index("deployment_connections_connection_idx").on(table.connectionId),
    index("deployment_connections_app_env_idx").on(table.appId, table.environmentId),
  ],
);
