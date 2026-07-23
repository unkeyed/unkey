import { relations } from "drizzle-orm";
import {
  bigint,
  boolean,
  index,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { appDockerSources } from "./app_docker_sources";
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
    name: varchar("name", { length: 256 }).notNull(),
    slug: varchar("slug", { length: 256 }).notNull(),

    // Where deployments for this app come from. `github` apps build from a
    // connected repo (`github_repo_connections`); `docker_image` apps deploy
    // a prebuilt image (`app_docker_sources`). Connecting a repo flips this
    // to `github`; disconnecting flips it back to `docker_image`.
    sourceType: mysqlEnum("source_type", ["github", "docker_image"])
      .notNull()
      .default("docker_image"),

    defaultBranch: varchar("default_branch", { length: 256 }).notNull().default("main"),
    currentDeploymentId: varchar("current_deployment_id", { length: 256 }),
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
  dockerSource: one(appDockerSources, {
    fields: [apps.id],
    references: [appDockerSources.appId],
  }),
}));
