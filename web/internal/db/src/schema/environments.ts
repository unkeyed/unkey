import { relations } from "drizzle-orm";
import { index, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { projects } from "./projects";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";
export const environments = mysqlTable(
  "environments",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),

    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    appId: id("app_id").notNull(),

    slug: varchar("slug", { length: 256 }).notNull(), // URL-safe identifier within workspace
    description: varchar("description", { length: 255 }).notNull().default(""),
    kind: mysqlEnum("kind", ["production", "preview"]).notNull().default("preview"),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("environments_app_slug_idx").on(table.appId, table.slug),
    index("environments_project_idx").on(table.projectId),
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
