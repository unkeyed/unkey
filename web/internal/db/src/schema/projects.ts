import { relations } from "drizzle-orm";
import { bigint, boolean, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { deployments } from "./deployments";
import { frontlineRoutes } from "./frontline_routes";
import { githubRepoConnections } from "./github_app";
export const projects = mysqlTable(
  "projects",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    name: varchar("name", { length: 256 }).notNull(),
    slug: varchar("slug", { length: 256 }).notNull(), // URL-safe identifier within workspace

    liveDeploymentId: varchar("live_deployment_id", { length: 256 }),
    isRolledBack: boolean("is_rolled_back").notNull().default(false),
    defaultBranch: varchar("default_branch", { length: 256 }).default("main"),
    depotProjectId: varchar("depot_project_id", { length: 255 }),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => [uniqueIndex("workspace_slug_idx").on(table.workspaceId, table.slug)],
);

export const projectsRelations = relations(projects, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [projects.workspaceId],
    references: [workspaces.id],
  }),
  apps: many(apps),
  deployments: many(deployments),
  frontlineRoutes: many(frontlineRoutes),
  githubRepoConnections: many(githubRepoConnections),
}));
