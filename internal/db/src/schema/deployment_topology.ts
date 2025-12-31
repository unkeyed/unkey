import { relations } from "drizzle-orm";
import {
  index,
  int,
  mysqlEnum,
  mysqlTable,
  primaryKey,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const deploymentTopology = mysqlTable(
  "deployment_topology",
  {
    workspaceId: varchar("workspace_id", { length: 64 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 64 }).notNull(),

    region: varchar("region", { length: 64 }).notNull(),

    desiredReplicas: int("desired_replicas").notNull(),

    // Deployment status
    desiredStatus: mysqlEnum("desired_status", [
      "starting",
      "started",
      "stopping",
      "stopped",
    ]).notNull(),
    ...lifecycleDates,
  },
  (table) => [
    primaryKey({ columns: [table.deploymentId, table.region] }),
    uniqueIndex("unique_region_per_deployment").on(table.deploymentId, table.region),
    index("workspace_idx").on(table.workspaceId),
    index("deployment_idx").on(table.deploymentId),
    index("region_idx").on(table.region),
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
