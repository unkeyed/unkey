import { relations } from "drizzle-orm";
import { bigint, index, json, mysqlEnum, mysqlTable, text, varchar } from "drizzle-orm/mysql-core";
import { deploymentSteps } from "./deployment_steps";
import { environments } from "./environments";
import { gateways } from "./gateways";
import { instances } from "./instances";
import { projects } from "./projects";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";
import { workspaces } from "./workspaces";

export const deployments = mysqlTable(
  "deployments",
  {
    id: varchar("id", { length: 128 }).primaryKey(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    // Environment configuration (production, preview, etc.)
    environmentId: varchar("environment_id", { length: 128 }).notNull(),

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

    // Immutable configuration snapshot
    runtimeConfig: json("runtime_config")
      .$type<{
        regions: Array<{ region: string; vmCount: number }>;
        cpus: number;
        memory: number;
      }>()
      .notNull(),

    gatewayConfig: longblob("gateway_config").notNull(),

    // OpenAPI specification
    openapiSpec: longblob("openapi_spec"),

    // Environment variables snapshot (protobuf: ctrl.v1.SecretsBlob)
    // Encrypted values from environment_variables at deploy time
    secretsConfig: longblob("secrets_config").notNull(),

    // Deployment status
    status: mysqlEnum("status", ["pending", "building", "deploying", "network", "ready", "failed"])
      .notNull()
      .default("pending"),
    ...lifecycleDates,
  },
  (table) => [
    index("workspace_idx").on(table.workspaceId),
    index("project_idx").on(table.projectId),
    index("status_idx").on(table.status),
  ],
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
  gateways: many(gateways),
  instances: many(instances),
}));
