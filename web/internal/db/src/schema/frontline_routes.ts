import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { projects } from "./projects";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";

export const frontlineRoutes = mysqlTable(
  "frontline_routes",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 128 }).notNull().unique(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    fullyQualifiedDomainName: varchar("fully_qualified_domain_name", {
      length: 256,
    })
      .notNull()
      .unique(),
    // sticky determines whether a fullyQualifiedDomainName should get reassigned to the latest deployment
    // - branch: the fullyQualifiedDomainName always points to the latest deployment on the branch
    //     <projectslug>-<appslug>-git-<branchname>-<workspaceslug>.unkey.app
    //
    // - environment: the fullyQualifiedDomainName is sticky to the environment it was created on
    //     <projectslug>-<appslug>-<environmentslug>-<workspaceslug>.unkey.app
    //
    // - live: the fullyQualifiedDomainName is sticky to the live deployment it was created on
    //     api.unkey.com
    //
    // - deployment: per-deployment stable URL, never reassigned
    //     <projectslug>-<appslug>-<id>-<workspaceslug>.unkey.app
    sticky: mysqlEnum("sticky", ["none", "branch", "environment", "live", "deployment"])
      .notNull()
      .default("none"),

    // Edge-redirect rules attached to this route. Stored as protojson of
    // frontline.edgeredirect.v1.Config. Empty/"{}" means no rules: the
    // request falls through to the proxy. Frontline parses this once at
    // cache fill time, never on the hot path.
    edgeRedirectConfig: longblob("edge_redirect_config").notNull(),

    ...lifecycleDates,
  },
  (table) => [
    index("project_id_idx").on(table.projectId),
    index("environment_id_idx").on(table.environmentId),
    index("deployment_id_idx").on(table.deploymentId),
    index("fqdn_environment_deployment_idx").on(
      table.fullyQualifiedDomainName,
      table.environmentId,
      table.deploymentId,
    ),
  ],
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
