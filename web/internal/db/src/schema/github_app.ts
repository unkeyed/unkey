import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { projects } from "./projects";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const githubAppInstallations = mysqlTable("github_app_installations", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
  installationId: bigint("installation_id", { mode: "number" }).notNull().unique(),
  ...lifecycleDates,
});

export const githubAppInstallationsRelations = relations(githubAppInstallations, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [githubAppInstallations.workspaceId],
    references: [workspaces.id],
  }),
}));

export const githubRepoConnections = mysqlTable(
  "github_repo_connections",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    projectId: varchar("project_id", { length: 64 }).notNull().unique(),
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
  project: one(projects, {
    fields: [githubRepoConnections.projectId],
    references: [projects.id],
  }),
  installation: one(githubAppInstallations, {
    fields: [githubRepoConnections.installationId],
    references: [githubAppInstallations.installationId],
  }),
}));
