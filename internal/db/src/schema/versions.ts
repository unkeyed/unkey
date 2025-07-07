import { relations } from "drizzle-orm";
import { index, json, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { branches } from "./branches";
import { builds } from "./builds";
import { projects } from "./projects";
import { rootfsImages } from "./rootfs_images";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const versions = mysqlTable(
  "versions",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    branchId: varchar("branch_id", { length: 256 }),

    // Build information
    buildId: varchar("build_id", { length: 256 }),
    rootfsImageId: varchar("rootfs_image_id", { length: 256 }).notNull(),

    // Git information
    gitCommitSha: varchar("git_commit_sha", { length: 40 }),
    gitBranch: varchar("git_branch", { length: 256 }),

    // Immutable configuration snapshot
    configSnapshot: json("config_snapshot")
      .$type<{
        // Resolved environment variables
        envVariables: Record<string, string>;
        // Any other configuration captured at version creation
        metadata?: Record<string, unknown>;
      }>()
      .notNull(),

    // Version status
    status: mysqlEnum("status", [
      "pending",
      "building",
      "deploying",
      "active",
      "failed",
      "archived",
    ])
      .notNull()
      .default("pending"),

    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
    branchIdx: index("branch_idx").on(table.branchId),
    statusIdx: index("status_idx").on(table.status),
    rootfsImageIdx: index("rootfs_image_idx").on(table.rootfsImageId),
  }),
);

export const versionsRelations = relations(versions, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [versions.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [versions.projectId],
    references: [projects.id],
  }),
  branch: one(branches, {
    fields: [versions.branchId],
    references: [branches.id],
  }),
  build: one(builds, {
    fields: [versions.buildId],
    references: [builds.id],
  }),
  rootfsImage: one(rootfsImages, {
    fields: [versions.rootfsImageId],
    references: [rootfsImages.id],
  }),
  // routes: many(routes),
  // hostnames: many(hostnames),
}));
