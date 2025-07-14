import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { projects } from "./projects";
import { versions } from "./versions";
import { workspaces } from "./workspaces";
export const branches = mysqlTable(
  "branches",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    name: varchar("name", { length: 256 }).notNull(), // Git branch name

    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
    projectNameIdx: uniqueIndex("project_name_idx").on(table.projectId, table.name),
  }),
);

export const branchesRelations = relations(branches, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [branches.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [branches.projectId],
    references: [projects.id],
  }),
  versions: many(versions),
}));
