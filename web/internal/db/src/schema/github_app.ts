import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
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
