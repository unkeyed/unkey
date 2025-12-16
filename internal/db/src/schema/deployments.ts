import { relations } from "drizzle-orm";
import {
  bigint,
  index,
  int,
  mysqlEnum,
  mysqlTable,
  text,
  varchar,
} from "drizzle-orm/mysql-core";
import { deploymentSteps } from "./deployment_steps";
import { environments } from "./environments";
import { instances } from "./instances";
import { projects } from "./projects";
import { sentinels } from "./sentinels";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";
import { workspaces } from "./workspaces";

export const deployments = mysqlTable(
  "deployments",
  {
    id: varchar("id", { length: 128 }).primaryKey(),
    k8sCrdName: varchar("k8s_crd_name", { length: 255 }).notNull().unique(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    // Environment configuration (production, preview, etc.)
    environmentId: varchar("environment_id", { length: 128 }).notNull(),
    // the docker image
    // null until the build is done
    image: varchar("image", { length: 256 }),

    // Git information
    gitCommitSha: varchar("git_commit_sha", { length: 40 }),
    gitBranch: varchar("git_branch", { length: 256 }),
    gitCommitMessage: text("git_commit_message"),
    gitCommitAuthorHandle: varchar("git_commit_author_handle", {
      length: 256,
    }),
    gitCommitAuthorAvatarUrl: varchar("git_commit_author_avatar_url", {
      length: 512,
    }),
    gitCommitTimestamp: bigint("git_commit_timestamp", { mode: "number" }), // Unix epoch milliseconds

    sentinelConfig: longblob("sentinel_config").notNull(),

    // OpenAPI specification
    openapiSpec: longblob("openapi_spec"),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),
    desiredState: mysqlEnum("desired_state", ["running", "standby", "archived"])
      .notNull()
      .default("running"),

    // Deployment status
    status: mysqlEnum("status", [
      "pending",
      "building",
      "deploying",
      "network",
      "ready",
      "failed",
    ])
      .notNull()
      .default("pending"),
    ...lifecycleDates,
  },
  (table) => [
    index("workspace_idx").on(table.workspaceId),
    index("project_idx").on(table.projectId),
    index("status_idx").on(table.status),
  ]
);

export const deploymentsRelations = relations(deployments, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [deployments.workspaceId],
    references: [workspaces.id],
  }),
  environment: one(environments, {
    fields: [deployments.environmentId],
    references: [environments.id],
  }),
  project: one(projects, {
    fields: [deployments.projectId],
    references: [projects.id],
  }),

  steps: many(deploymentSteps),
  sentinels: many(sentinels),
  instances: many(instances),
}));
