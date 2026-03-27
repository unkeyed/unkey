import { relations } from "drizzle-orm";
import {
  bigint,
  index,
  int,
  mysqlEnum,
  mysqlTable,
  tinyint,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
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

    // HPA scaling configuration, snapshotted from the autoscaling policy at deploy time.
    // Minimum number of pod replicas the HPA will maintain.
    autoscalingReplicasMin: int("autoscaling_replicas_min").notNull().default(1),
    // Maximum number of pod replicas the HPA can scale to.
    autoscalingReplicasMax: int("autoscaling_replicas_max").notNull().default(1),
    // Average CPU utilization percentage (0-100) that triggers scale-up. Null = use default (80%).
    autoscalingThresholdCpu: tinyint("autoscaling_threshold_cpu"),
    // Average memory utilization percentage (0-100) that triggers scale-up. Null = not used as a signal.
    autoscalingThresholdMemory: tinyint("autoscaling_threshold_memory"),

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
    uniqueIndex("unique_region_per_deployment").on(table.deploymentId, table.regionId),
    uniqueIndex("unique_version_per_region").on(table.regionId, table.version),
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
