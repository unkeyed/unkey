import { index, mysqlEnum, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

// Chooses which deployment of the target app a connection default reaches, one
// row per app connection. Only app targets need a selection, so these columns
// live here instead of as nullable columns on appConnections. A deployment
// pinned here is kept running.
export const connectionAppTargets = mysqlTable(
  "connection_app_targets",
  {
    pk: primaryKey(),
    connectionId: id("connection_id").notNull(),
    selectionMode: mysqlEnum("selection_mode", [
      "automatic",
      "environment",
      "deployment",
    ]).notNull(),
    targetEnvironmentId: id("target_environment_id"),
    targetDeploymentId: id("target_deployment_id"),
  },
  (table) => [
    uniqueIndex("connection_app_targets_connection_idx").on(table.connectionId),
    index("connection_app_targets_deployment_idx").on(table.targetDeploymentId),
  ],
);
