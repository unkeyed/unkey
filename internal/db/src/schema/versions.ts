import { relations } from "drizzle-orm";
import { index, json, mysqlEnum, mysqlTable, text, varchar } from "drizzle-orm/mysql-core";
import { branches } from "./branches";
import { builds } from "./builds";
import { environments } from "./environments";
import { projects } from "./projects";
import { rootfsImages } from "./rootfs_images";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const versions = mysqlTable(
  "versions",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    environmentId: varchar("environment_id", { length: 256 }).notNull(),
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

    // Topology configuration
    topologyConfig: json("topology_config")
      .$type<{
        // Resource allocation
        cpuMillicores: number;
        memoryMb: number;

        // Auto-scaling configuration
        minInstances: number;
        maxInstances: number;

        // Regional deployment settings
        regions: Array<{
          region: string;
          minInstances: number;
          maxInstances: number;
        }>;

        // Timeouts
        idleTimeoutSeconds: number;
        gracefulShutdownSeconds: number;

        // Health check configuration
        healthCheck?: {
          path: string;
          intervalSeconds: number;
          timeoutSeconds: number;
          unhealthyThreshold: number;
        };
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

    // JWT signing keys for this version
    jwtPrivateKey: text("jwt_private_key"), // Encrypted
    jwtPublicKey: text("jwt_public_key"),

    ...lifecycleDatesMigration,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
    environmentIdx: index("environment_idx").on(table.environmentId),
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
  environment: one(environments, {
    fields: [versions.environmentId],
    references: [environments.id],
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
