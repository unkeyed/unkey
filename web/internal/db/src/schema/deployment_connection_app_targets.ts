import { index, mysqlEnum, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

// The selection rule saved with each deploymentConnections row, copied from
// connectionAppTargets at creation. Existing source deployments keep this rule
// when the default changes; saved automatic and environment rules are still
// resolved when read, as described on connectionAppTargets. A deployment pin
// protects its target from automatic retirement while the source deployment
// (the caller, never an end user) runs with private networking.
export const deploymentConnectionAppTargets = mysqlTable(
  "deployment_connection_app_targets",
  {
    pk: primaryKey(),
    deploymentId: id("deployment_id").notNull(),
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
    uniqueIndex("deployment_connection_app_targets_deployment_connection_idx").on(
      table.deploymentId,
      table.connectionId,
    ),
    index("deployment_connection_app_targets_target_deployment_idx").on(table.targetDeploymentId),
  ],
);
