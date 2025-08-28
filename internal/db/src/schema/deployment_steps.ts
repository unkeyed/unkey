import { relations } from "drizzle-orm";
import { bigint, mysqlEnum, mysqlTable, primaryKey, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";

export const deploymentSteps = mysqlTable(
  "deployment_steps",
  {
    deploymentId: varchar("deployment_id", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    status: mysqlEnum("status", [
      "pending",
      "downloading_docker_image",
      "building_rootfs",
      "uploading_rootfs",
      "creating_vm",
      "booting_vm",
      "assigning_domains",
      "completed",
      "failed",
    ]).notNull(),
    message: varchar("message", { length: 1024 }).notNull(),
    createdAt: bigint("crated_at", { mode: "number" }).notNull(),
  },
  (table) => ({
    pk: primaryKey({ columns: [table.deploymentId, table.status] }),
  }),
);

export const deploymentStepsRelations = relations(deploymentSteps, ({ one }) => ({
  deployment: one(deployments, {
    fields: [deploymentSteps.deploymentId],
    references: [deployments.id],
  }),
  // routes: many(routes),
  // hostnames: many(hostnames),
}));
