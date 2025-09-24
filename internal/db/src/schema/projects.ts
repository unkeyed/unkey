import { relations } from "drizzle-orm";
import { index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { deployments } from "./deployments";
export const projects = mysqlTable(
  "projects",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    name: varchar("name", { length: 256 }).notNull(),
    slug: varchar("slug", { length: 256 }).notNull(), // URL-safe identifier within workspace

    // Git configuration
    gitRepositoryUrl: varchar("git_repository_url", { length: 500 }),
    // this is likely temporary but we need a way to point to the current prod deployment.
    // in the future I think we want to have a special deployment per environment, but for now this is fine
    liveDeploymentId: varchar("live_deployment_id", { length: 256 }),
    rolledBackDeploymentId: varchar("rolled_back_deployment_id", {
      length: 256,
    }),

    defaultBranch: varchar("default_branch", { length: 256 }).default("main"),
    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    workspaceSlugIdx: uniqueIndex("workspace_slug_idx").on(table.workspaceId, table.slug),
  }),
);

export const projectsRelations = relations(projects, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [projects.workspaceId],
    references: [workspaces.id],
  }),
  deployments: many(deployments),
  activeDeployment: one(deployments, {
    fields: [projects.liveDeploymentId],
    references: [deployments.id],
  }),
  // environments: many(projectEnvironments),
}));
