import { relations } from "drizzle-orm";
import { mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { projects } from "./projects";
export const environments = mysqlTable(
  "environments",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    slug: varchar("slug", { length: 256 }).notNull(), // URL-safe identifier within workspace
    description: varchar("description", { length: 255 }),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => ({
    uniqueSlug: uniqueIndex("environments_project_id_slug_idx").on(table.projectId, table.slug),
  }),
);

export const environmentsRelations = relations(environments, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [environments.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [environments.projectId],
    references: [projects.id],
  }),
}));
