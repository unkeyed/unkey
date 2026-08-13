import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, text, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { projects } from "./projects";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const githubAppInstallations = mysqlTable(
  "github_app_installations",
  {
    pk: primaryKey(),
    workspaceId: id("workspace_id").notNull(),
    installationId: bigint("installation_id", { mode: "number" }).notNull(),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("workspace_installation_idx").on(table.workspaceId, table.installationId),
  ],
);

export const githubAppInstallationsRelations = relations(githubAppInstallations, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [githubAppInstallations.workspaceId],
    references: [workspaces.id],
  }),
}));

export const githubRepoConnections = mysqlTable(
  "github_repo_connections",
  {
    pk: primaryKey(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    appId: id("app_id").notNull().unique(),
    installationId: bigint("installation_id", {
      mode: "number",
    }).notNull(),
    repositoryId: bigint("repository_id", { mode: "number" }).notNull(),
    repositoryFullName: varchar("repository_full_name", {
      length: 500,
    }).notNull(),
    // Which SCM the connection belongs to. GitLab connections reuse this
    // table: installation_id and repository_id both hold the GitLab project id.
    provider: varchar("provider", { length: 32 }).notNull().default("github"),
    // POC only: plaintext clone credential (GitLab access token / Bitbucket
    // refresh token). Text, not varchar: Atlassian refresh tokens exceed 512
    // chars. Must move to vault-backed encrypted storage before this leaves
    // local development.
    accessToken: text("access_token"),
    ...lifecycleDates,
  },
  (table) => [index("installation_id_idx").on(table.installationId)],
);

export const githubRepoConnectionsRelations = relations(githubRepoConnections, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [githubRepoConnections.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [githubRepoConnections.projectId],
    references: [projects.id],
  }),
  app: one(apps, {
    fields: [githubRepoConnections.appId],
    references: [apps.id],
  }),
  installation: one(githubAppInstallations, {
    fields: [githubRepoConnections.installationId],
    references: [githubAppInstallations.installationId],
  }),
}));
