import { relations } from "drizzle-orm";
import { boolean, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { environments } from "./environments";
import { projects } from "./projects";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const branches = mysqlTable(
  "branches",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    name: varchar("name", { length: 256 }).notNull(), // Git branch name
    environmentId: varchar("environment_id", { length: 256 }).notNull(), // Which environment this branch deploys to

    // Is this the main/production branch for the project
    isProduction: boolean("is_production").notNull().default(false),

    ...lifecycleDatesMigration,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
    environmentIdx: index("environment_idx").on(table.environmentId),
    projectNameIdx: uniqueIndex("project_name_idx").on(table.projectId, table.name),
  }),
);

export const branchesRelations = relations(branches, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [branches.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [branches.projectId],
    references: [projects.id],
  }),
  environment: one(environments, {
    fields: [branches.environmentId],
    references: [environments.id],
  }),
  // versions: many(versions),
}));
