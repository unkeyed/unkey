import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { projects } from "./projects";
export const environments = mysqlTable(
  "environments",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 128 }).notNull().unique(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),

    slug: varchar("slug", { length: 256 }).notNull(), // URL-safe identifier within workspace
    description: varchar("description", { length: 255 }).notNull().default(""),

    // FK to deletions.id. NULL means the environment is live; non-NULL
    // means the environment is in its soft-delete grace window. Set by
    // EnvironmentService.MarkForDeletion (cascaded from app/project)
    // and cleared by EnvironmentService.Restore.
    deletionId: varchar("deletion_id", { length: 64 }),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("environments_app_slug_idx").on(table.appId, table.slug),
    index("environments_project_idx").on(table.projectId),
    index("environments_deletion_id_idx").on(table.deletionId),
  ],
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
  app: one(apps, {
    fields: [environments.appId],
    references: [apps.id],
  }),
}));
