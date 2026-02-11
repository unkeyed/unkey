import { relations, sql } from "drizzle-orm";
import {
  bigint,
  index,
  int,
  json,
  mysqlEnum,
  mysqlTable,
  text,
  varchar,
} from "drizzle-orm/mysql-core";
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
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 128 }).notNull().unique(),
    k8sName: varchar("k8s_name", { length: 255 }).notNull().unique(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    // Environment configuration (production, preview, etc.)
    environmentId: varchar("environment_id", { length: 128 }).notNull(),

    // the docker image
    // null until the build is done
    image: varchar("image", { length: 256 }),
    buildId: varchar("build_id", { length: 128 }).unique(),

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

    // Environment variables snapshot (protobuf: ctrl.v1.SecretsBlob)
    // Encrypted values from environment_variables at deploy time
    encryptedEnvironmentVariables: longblob("encrypted_environment_variables").notNull(),

    // Container command override (e.g., ["./app", "serve"])
    // If empty, the container's default entrypoint/cmd is used
    command: json("command").$type<string[]>().notNull().default(sql`('[]')`),

    // Port the container listens on
    port: int("port").notNull().default(8080),

    // Signal sent to the container for graceful shutdown
    shutdownSignal: mysqlEnum("shutdown_signal", ["SIGTERM", "SIGINT", "SIGQUIT", "SIGKILL"])
      .notNull()
      .default("SIGTERM"),

    // HTTP healthcheck configuration (null = no healthcheck)
    healthcheck: json("healthcheck").$type<import("./environment_runtime_settings").Healthcheck>(),

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

  sentinels: many(sentinels),
  instances: many(instances),
}));
