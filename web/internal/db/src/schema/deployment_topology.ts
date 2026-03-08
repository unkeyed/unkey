import { relations } from "drizzle-orm";
import { bigint, index, int, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const deploymentTopology = mysqlTable(
  "deployment_topology",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 64 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 64 }).notNull(),
    regionId: varchar("region_id", { length: 64 }).notNull(),

    desiredReplicas: int("desired_replicas").notNull(),

    // Version for state synchronization with edge agents.
    // Updated via Restate VersioningService on each mutation.
    // Edge agents track their last-seen version and request changes after it.
    // Unique per regionId (composite index with regionId).
    version: bigint("version", { mode: "number", unsigned: true }).notNull(),

    // Deployment status
    desiredStatus: mysqlEnum("desired_status", ["stopped", "running"]).notNull(),
    ...lifecycleDates,
  },
  (table) => [
    //uniqueIndex("unique_region_per_deployment").on(table.deploymentId, table.regionId),
    //uniqueIndex("unique_version_per_region").on(table.regionId, table.version),
    //uniqueIndex("deployment_region_idx").on(table.deploymentId, table.regionId),
    index("workspace_idx").on(table.workspaceId),
    index("status_idx").on(table.desiredStatus),
  ],
);

export const deploymentTopologyRelations = relations(deploymentTopology, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [deploymentTopology.workspaceId],
    references: [workspaces.id],
  }),
  delpoyment: one(deployments, {
    fields: [deploymentTopology.deploymentId],
    references: [deployments.id],
  }),
}));
