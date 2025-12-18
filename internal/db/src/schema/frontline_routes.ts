import { relations } from "drizzle-orm";
import { index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { projects } from "./projects";
import { lifecycleDates } from "./util/lifecycle_dates";

export const frontlineRoutes = mysqlTable(
  "frontline_routes",
  {
    id: varchar("id", { length: 128 }).primaryKey(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    fullyQualifiedDomainName: varchar("fully_qualified_domain_name", {
      length: 256,
    })
      .notNull()
      .unique(),
    // sticky determines whether a fullyQualifiedDomainName should get reassigned to the latest deployment
    // - branch: the fullyQualifiedDomainName always points to the latest deployment on the branch
    //     <projectslug>-git-<branchname>-<workspaceslug>.unkey.app
    //
    // - environment: the fullyQualifiedDomainName is sticky to the environment it was created on
    //     <projectslug>-<environmentslug>-<workspaceslug>.unkey.app
    //
    // - live: the fullyQualifiedDomainName is sticky to the live deployment it was created on
    //     api.unkey.com
    sticky: mysqlEnum("sticky", ["none", "branch", "environment", "live"])
      .notNull()
      .default("none"),

    ...lifecycleDates,
  },
  (table) => [
    index("environment_id_idx").on(table.environmentId),
    index("deployment_id_idx").on(table.deploymentId),
  ]
);

export const frontlineRelations = relations(frontlineRoutes, ({ one }) => ({
  deployment: one(deployments, {
    fields: [frontlineRoutes.deploymentId],
    references: [deployments.id],
  }),
  project: one(projects, {
    fields: [frontlineRoutes.projectId],
    references: [projects.id],
  }),
}));
