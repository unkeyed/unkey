import { relations } from "drizzle-orm";
import {
  boolean,
  index,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { appSourceOci } from "./app_source_oci";
import { environments } from "./environments";
import { githubRepoConnections } from "./github_app";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { projects } from "./projects";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const apps = mysqlTable(
  "apps",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    name: varchar("name", { length: 256 }).notNull(),
    slug: varchar("slug", { length: 256 }).notNull(),
    sourceType: mysqlEnum("source_type", ["unknown", "git", "oci"]).notNull().default("unknown"),

    currentDeploymentId: id("current_deployment_id"),
    isRolledBack: boolean("is_rolled_back").notNull().default(false),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("apps_project_slug_idx").on(table.projectId, table.slug),
    index("apps_workspace_idx").on(table.workspaceId),
  ],
);

export const appsRelations = relations(apps, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [apps.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [apps.projectId],
    references: [projects.id],
  }),
  environments: many(environments),
  githubRepoConnection: one(githubRepoConnections, {
    fields: [apps.id],
    references: [githubRepoConnections.appId],
  }),
  ociSource: one(appSourceOci, {
    fields: [apps.id],
    references: [appSourceOci.appId],
  }),
}));
