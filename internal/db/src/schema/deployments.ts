import { relations } from "drizzle-orm";
import { index, json, mysqlEnum, mysqlTable, text, varchar } from "drizzle-orm/mysql-core";
import { deploymentSteps } from "./deployment_steps";
import { environments } from "./environments";
import { projects } from "./projects";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const deployments = mysqlTable(
  "deployments",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    // Environment configuration (production, preview, etc.)
    environmentId: varchar("environment", { length: 256 }).notNull(),

    // Git information
    gitCommitSha: varchar("git_commit_sha", { length: 40 }),
    gitBranch: varchar("git_branch", { length: 256 }),

    // Immutable configuration snapshot
    runtimeConfig: json("runtime_config")
      .$type<{
        regions: Array<{ region: string; vmCount: number }>;
        cpus: number;
        memory: number;
      }>()
      .notNull(),

    // OpenAPI specification
    openapiSpec: text("openapi_spec"),

    // Deployment status
    status: mysqlEnum("status", ["pending", "building", "deploying", "ready", "failed"])
      .notNull()
      .default("pending"),
    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
    statusIdx: index("status_idx").on(table.status),
  }),
);

export const deploymentsRelations = relations(deployments, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [deployments.workspaceId],
    references: [workspaces.id],
  }),
  environment: one(environments, {
    fields: [deployments.environmentId],
    references: [environments.id],
  }),
  project: one(projects, {
    fields: [deployments.projectId],
    references: [projects.id],
  }),

  steps: many(deploymentSteps),
}));
