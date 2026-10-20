import { index, mysqlEnum, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

// The selection rule of one app connection, exactly one row per appConnections
// row. It is not a second connection: it picks which deployment of the
// connection's target app (Y) the source reaches.
//
// automatic follows the target app: production sources reach its current
// deployment, preview sources its newest ever-ready preview deployment from
// the same git branch and fork. environment follows target_environment_id.
// Both are resolved again when read. deployment pins target_deployment_id and protects that
// deployment from automatic retirement.
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
