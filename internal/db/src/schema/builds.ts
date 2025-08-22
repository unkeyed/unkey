import { relations } from "drizzle-orm";
import {
  bigint,
  index,
  mysqlEnum,
  mysqlTable,
  primaryKey,
  text,
  varchar,
} from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { projects } from "./projects";
import { rootfsImages } from "./rootfs_images";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";
export const builds = mysqlTable(
  "builds",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 256 }).notNull(),

    // Build result
    rootfsImageId: varchar("rootfs_image_id", { length: 256 }), // Null until build completes

    // Git information
    gitCommitSha: varchar("git_commit_sha", { length: 40 }),
    gitBranch: varchar("git_branch", { length: 256 }),

    // Build status
    status: mysqlEnum("status", ["pending", "running", "succeeded", "failed", "cancelled"])
      .notNull()
      .default("pending"),

    // Build configuration
    buildTool: mysqlEnum("build_tool", ["docker", "depot", "custom"]).notNull().default("docker"),

    // Error tracking
    errorMessage: text("error_message"),

    // Timing
    startedAt: bigint("started_at", { mode: "number" }),
    completedAt: bigint("completed_at", { mode: "number" }),

    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
    statusIdx: index("status_idx").on(table.status),
    rootfsImageIdx: index("rootfs_image_idx").on(table.rootfsImageId),
  }),
);

export const deploymentSteps = mysqlTable(
  "deployment_steps",
  {
    deploymentId: varchar("deployment_id", { length: 256 }).notNull(),
    status: mysqlEnum("status", [
      "pending",
      "downloading_docker_image",
      "building_rootfs",
      "uploading_rootfs",
      "creating_vm",
      "booting_vm",
      "assigning_domains",
      "completed",
      "failed",
    ]).notNull(),
    message: text("message"),
    errorMessage: text("error_message"),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
  },
  (table) => ({
    pk: primaryKey({ columns: [table.deploymentId, table.status] }),
    deploymentIdCreatedAtIdx: index("idx_deployment_id_created_at").on(
      table.deploymentId,
      table.createdAt,
    ),
  }),
);

export const buildsRelations = relations(builds, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [builds.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [builds.projectId],
    references: [projects.id],
  }),
  rootfsImage: one(rootfsImages, {
    fields: [builds.rootfsImageId],
    references: [rootfsImages.id],
  }),
  deployment: one(deployments),
}));

export const deploymentStepsRelations = relations(deploymentSteps, ({ one }) => ({
  deployment: one(deployments, {
    fields: [deploymentSteps.deploymentId],
    references: [deployments.id],
  }),
}));
