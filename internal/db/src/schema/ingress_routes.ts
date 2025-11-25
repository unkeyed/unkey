import { relations } from "drizzle-orm";
import { index, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { projects } from "./projects";
import { lifecycleDates } from "./util/lifecycle_dates";

export const ingressRoutes = mysqlTable(
  "ingress_routes",
  {
    id: varchar("id", { length: 128 }).primaryKey(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    hostname: varchar("hostname", { length: 256 }).notNull(),
    // sticky determines whether a hostname should get reassigned to the latest deployment
    // - branch: the hostname always points to the latest deployment on the branch
    //     <projectslug>-git-<branchname>-<workspaceslug>.unkey.app
    //
    // - environment: the hostname is sticky to the environment it was created on
    //     <projectslug>-<environmentslug>-<workspaceslug>.unkey.app
    //
    // - live: the hostname is sticky to the live deployment it was created on
    //     api.unkey.com
    sticky: mysqlEnum("sticky", ["none", "branch", "environment", "live"])
      .notNull()
      .default("none"),

    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("unique_hostname_idx").on(table.hostname),
    index("environment_id_idx").on(table.environmentId),
    index("deployment_id_idx").on(table.deploymentId),
  ],
);

export const ingressRelations = relations(ingressRoutes, ({ one }) => ({
  deployment: one(deployments, {
    fields: [ingressRoutes.deploymentId],
    references: [deployments.id],
  }),
  project: one(projects, {
    fields: [ingressRoutes.projectId],
    references: [projects.id],
  }),
}));
