import { relations } from "drizzle-orm";
import { bigint, boolean, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { environments } from "./environments";
import { githubRepoConnections } from "./github_app";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { projects } from "./projects";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";

export const apps = mysqlTable(
  "apps",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: caseSensitiveVarchar("id", { length: 32 }).notNull().unique(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 32 }).notNull(),
    projectId: caseSensitiveVarchar("project_id", { length: 32 }).notNull(),
    name: varchar("name", { length: 256 }).notNull(),
    slug: varchar("slug", { length: 256 }).notNull(),

    defaultBranch: caseSensitiveVarchar("default_branch", { length: 256 })
      .notNull()
      .default("main"),
    currentDeploymentId: caseSensitiveVarchar("current_deployment_id", { length: 32 }),
    isRolledBack: boolean("is_rolled_back").notNull().default(false),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("apps_project_slug_idx").on(table.projectId, table.slug),
    index("apps_workspace_idx").on(table.workspaceId),
  ],
);

export const appsRelations = relations(apps, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [apps.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [apps.projectId],
    references: [projects.id],
  }),
  environments: many(environments),
  githubRepoConnection: one(githubRepoConnections, {
    fields: [apps.id],
    references: [githubRepoConnections.appId],
  }),
}));
