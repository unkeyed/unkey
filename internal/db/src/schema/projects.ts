import { relations } from "drizzle-orm";
import { index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { partitions } from "./partitions";
import { versions } from "./versions";
export const projects = mysqlTable(
  "projects",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    partitionId: varchar("partition_id", { length: 256 }).notNull(),

    name: varchar("name", { length: 256 }).notNull(),
    slug: varchar("slug", { length: 256 }).notNull(), // URL-safe identifier within workspace

    // Git configuration
    gitRepositoryUrl: varchar("git_repository_url", { length: 500 }),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    partitionIdx: index("partition_idx").on(table.partitionId),
    workspaceSlugIdx: uniqueIndex("workspace_slug_idx").on(table.workspaceId, table.slug),
  }),
);

export const projectsRelations = relations(projects, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [projects.workspaceId],
    references: [workspaces.id],
  }),
  partition: one(partitions, {
    fields: [projects.partitionId],
    references: [partitions.id],
  }),
  // branches: many(branches),
  versions: many(versions),
  // environments: many(projectEnvironments),
}));
