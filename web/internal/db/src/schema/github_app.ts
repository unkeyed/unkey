import { relations } from "drizzle-orm";
import { bigint, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { projects } from "./projects";
import { lifecycleDates } from "./util/lifecycle_dates";

export const githubAppInstallations = mysqlTable("github_app_installations", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  id: varchar("id", { length: 256 }).notNull().unique(),
  projectId: varchar("project_id", { length: 64 }).notNull().unique(),
  installationId: bigint("installation_id", { mode: "number" }).notNull(),
  repositoryId: bigint("repository_id", { mode: "number" }).notNull(),
  repositoryFullName: varchar("repository_full_name", {
    length: 500,
  }).notNull(),
  ...lifecycleDates,
});

export const githubAppInstallationsRelations = relations(githubAppInstallations, ({ one }) => ({
  project: one(projects, {
    fields: [githubAppInstallations.projectId],
    references: [projects.id],
  }),
}));
