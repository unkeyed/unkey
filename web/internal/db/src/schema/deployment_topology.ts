import { relations } from "drizzle-orm";
import {
  bigint,
  index,
  int,
  mysqlEnum,
  mysqlTable,
  tinyint,
  uniqueIndex,
} from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
// import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const deploymentTopology = mysqlTable(
  "deployment_topology",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    // workspaceId: id("workspace_id").notNull(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 64 }).notNull(),
    // deploymentId: id("deployment_id").notNull(),
    deploymentId: caseSensitiveVarchar("deployment_id", { length: 64 }).notNull(),
    // regionId: id("region_id").notNull(),
    regionId: caseSensitiveVarchar("region_id", { length: 64 }).notNull(),

    // HPA scaling configuration, snapshotted from the autoscaling policy at deploy time.
    // Minimum number of pod replicas the HPA will maintain.
    autoscalingReplicasMin: int("autoscaling_replicas_min", { unsigned: true })
      .notNull()
      .default(1),
    // Maximum number of pod replicas the HPA can scale to.
    autoscalingReplicasMax: int("autoscaling_replicas_max", { unsigned: true })
      .notNull()
      .default(1),
    // Average CPU utilization percentage (0-100) that triggers scale-up. Null = use default (80%).
    autoscalingThresholdCpu: tinyint("autoscaling_threshold_cpu", { unsigned: true }),
    // Average memory utilization percentage (0-100) that triggers scale-up. Null = not used as a signal.
    autoscalingThresholdMemory: tinyint("autoscaling_threshold_memory", { unsigned: true }),

    // Deployment status
    desiredStatus: mysqlEnum("desired_status", ["stopped", "running"]).notNull(),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("unique_region_per_deployment").on(table.deploymentId, table.regionId),
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
