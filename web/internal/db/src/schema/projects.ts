import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { deployments } from "./deployments";
import { environments } from "./environments";
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

    depotProjectId: varchar("depot_project_id", { length: 255 }),

    // FK to deletions.id. NULL means the project is live; non-NULL means
    // the project is in its soft-delete grace window and the deletions
    // row holds the cascade timestamp T. Find/List queries filter on
    // NULL so a scheduled project is invisible to the API.
    deletionId: varchar("deletion_id", { length: 64 }),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("workspace_slug_idx").on(table.workspaceId, table.slug),
    index("projects_deletion_id_idx").on(table.deletionId),
  ],
);

export const projectsRelations = relations(projects, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [projects.workspaceId],
    references: [workspaces.id],
  }),
  environments: many(environments),
  apps: many(apps),
  deployments: many(deployments),
  frontlineRoutes: many(frontlineRoutes),
  githubRepoConnections: many(githubRepoConnections),
}));
