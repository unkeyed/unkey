import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { environments } from "./environments";
import { projects } from "./projects";
import { workspaces } from "./workspaces";

export const deploymentSteps = mysqlTable(
  "deployment_steps",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 128 }).notNull(),
    projectId: varchar("project_id", { length: 128 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 128 }).notNull(),

    step: mysqlEnum("step", ["queued", "building", "deploying", "network"])
      .notNull()
      .default("queued"),

    startedAt: bigint("started_at", {
      mode: "number",
      unsigned: true,
    }).notNull(),
    endedAt: bigint("ended_at", { mode: "number", unsigned: true }),
    error: varchar("error", { length: 512 }),
  },
  (table) => [
    index("workspace_idx").on(table.workspaceId),
    index("deployment_idx").on(table.deploymentId),
    uniqueIndex("unique_step_per_deployment").on(table.deploymentId, table.step),
  ],
);

export const deploymentStepsRelations = relations(deploymentSteps, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [deploymentSteps.workspaceId],
    references: [workspaces.id],
  }),
  environment: one(environments, {
    fields: [deploymentSteps.environmentId],
    references: [environments.id],
  }),
  project: one(projects, {
    fields: [deploymentSteps.projectId],
    references: [projects.id],
  }),
  deployment: one(deployments, {
    fields: [deploymentSteps.deploymentId],
    references: [deployments.id],
  }),
}));
