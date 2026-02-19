import { relations } from "drizzle-orm";
import { bigint, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
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

    defaultBranch: varchar("default_branch", { length: 256 }).default("main"),

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
  githubRepoConnection: one(githubRepoConnections, {
    fields: [projects.id],
    references: [githubRepoConnections.projectId],
  }),
}));
