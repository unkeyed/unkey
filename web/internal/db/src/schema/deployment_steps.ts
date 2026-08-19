import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { environments } from "./environments";
import { projects } from "./projects";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const deploymentSteps = mysqlTable(
  "deployment_steps",
  {
    pk: primaryKey(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    environmentId: id("environment_id").notNull(),
    deploymentId: id("deployment_id").notNull(),
    appId: id("app_id").notNull(),

    step: mysqlEnum("step", [
      "queued",
      "starting",
      "building",
      "deploying",
      "network",
      "finalizing",
    ])
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
    uniqueIndex("unique_step_per_deployment").on(table.deploymentId, table.step),
    // Serves the step-duration average: WHERE step = ? AND started_at >= ?
    // reading ended_at. All three columns in the index keep it index-only.
    index("step_started_at_idx").on(table.step, table.startedAt, table.endedAt),
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
