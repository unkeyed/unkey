import { bigint, index, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

// Immutable copy of the connection defaults a deployment was created with. Its
// host variables were derived from these names, so routing keeps using them
// after the defaults are renamed or removed. Automatic and environment targets
// still resolve to the target app's current deployment when read.
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
    resourceType: caseSensitiveVarchar("resource_type", { length: 32 }).notNull(),
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
