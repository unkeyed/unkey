import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { projects } from "./projects";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
export const environments = mysqlTable(
  "environments",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: caseSensitiveVarchar("id", { length: 128 }).notNull().unique(),

    workspaceId: caseSensitiveVarchar("workspace_id", { length: 256 }).notNull(),
    projectId: caseSensitiveVarchar("project_id", { length: 256 }).notNull(),
    appId: caseSensitiveVarchar("app_id", { length: 64 }).notNull(),

    slug: varchar("slug", { length: 256 }).notNull(), // URL-safe identifier within workspace
    description: varchar("description", { length: 255 }).notNull().default(""),

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
