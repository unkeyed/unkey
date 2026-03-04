import { relations } from "drizzle-orm";
import { bigint, boolean, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { environments } from "./environments";
import { githubRepoConnections } from "./github_app";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { projects } from "./projects";

export const apps = mysqlTable(
  "apps",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull().default(""),

    name: varchar("name", { length: 256 }).notNull(),
    slug: varchar("slug", { length: 256 }).notNull(),

    currentDeploymentId: varchar("current_deployment_id", { length: 256 }),
    isRolledBack: boolean("is_rolled_back").notNull().default(false),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("apps_env_slug_idx").on(table.environmentId, table.slug),
    index("apps_workspace_idx").on(table.workspaceId),
    index("apps_project_idx").on(table.projectId),
  ],
);

export const appsRelations = relations(apps, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [apps.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [apps.projectId],
    references: [projects.id],
  }),
  environment: one(environments, {
    fields: [apps.environmentId],
    references: [environments.id],
  }),
  githubRepoConnection: one(githubRepoConnections, {
    fields: [apps.id],
    references: [githubRepoConnections.appId],
  }),
}));
