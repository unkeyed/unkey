import { relations } from "drizzle-orm";
import { bigint, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { projects } from "./projects";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";

export const githubAppInstallations = mysqlTable("github_app_installations", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  id: varchar("id", { length: 256 }).notNull().unique(),
  projectId: varchar("project_id", { length: 64 }).notNull().unique(),
  installationId: varchar("installation_id", { length: 256 }).notNull(),
  repositoryId: varchar("repository_id", { length: 256 }).notNull(),
  repositoryFullName: varchar("repository_full_name", { length: 500 }).notNull(),
  ...lifecycleDatesMigration,
});

export const githubAppInstallationsRelations = relations(githubAppInstallations, ({ one }) => ({
  project: one(projects, {
    fields: [githubAppInstallations.projectId],
    references: [projects.id],
  }),
}));
