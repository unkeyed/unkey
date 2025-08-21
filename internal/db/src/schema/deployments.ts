import { relations } from "drizzle-orm";
import { bigint, index, json, mysqlEnum, mysqlTable, text, varchar } from "drizzle-orm/mysql-core";
import { builds } from "./builds";
import { projects } from "./projects";
import { rootfsImages } from "./rootfs_images";
import { workspaces } from "./workspaces";

export const deployments = mysqlTable(
  "deployments",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    // Environment configuration (production, preview, etc.)
    environment: mysqlEnum("environment", ["production", "preview"]).notNull().default("preview"),

    // Build information
    buildId: varchar("build_id", { length: 256 }),
    rootfsImageId: varchar("rootfs_image_id", { length: 256 }).notNull(),

    // Git information
    gitCommitSha: varchar("git_commit_sha", { length: 40 }),
    gitBranch: varchar("git_branch", { length: 256 }),
    gitCommitMessage: text("git_commit_message"),
    gitCommitAuthorName: varchar("git_commit_author_name", { length: 256 }),
    gitCommitAuthorEmail: varchar("git_commit_author_email", { length: 256 }),
    gitCommitAuthorUsername: varchar("git_commit_author_username", { length: 256 }),
    gitCommitAuthorAvatarUrl: varchar("git_commit_author_avatar_url", { length: 512 }),
    gitCommitTimestamp: bigint("git_commit_timestamp", { mode: "number" }), // Unix epoch milliseconds

    // Immutable configuration snapshot
    configSnapshot: json("config_snapshot")
      .$type<{
        // Resolved environment variables
        envVariables: Record<string, string>;
        // Any other configuration captured at version creation
        metadata?: Record<string, unknown>;
      }>()
      .notNull(),

    // OpenAPI specification
    openapiSpec: text("openapi_spec"),

    // Deployment status
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

    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    updatedAt: bigint("updated_at", { mode: "number" }),
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
    environmentIdx: index("environment_idx").on(table.environment),
    statusIdx: index("status_idx").on(table.status),
    rootfsImageIdx: index("rootfs_image_idx").on(table.rootfsImageId),
  }),
);

export const deploymentsRelations = relations(deployments, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [deployments.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [deployments.projectId],
    references: [projects.id],
  }),
  build: one(builds, {
    fields: [deployments.buildId],
    references: [builds.id],
  }),
  rootfsImage: one(rootfsImages, {
    fields: [deployments.rootfsImageId],
    references: [rootfsImages.id],
  }),
  // routes: many(routes),
  // hostnames: many(hostnames),
}));
