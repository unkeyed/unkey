import {
  boolean,
  index,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";

export const domains = mysqlTable(
  "domains",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }),
    deploymentId: varchar("deployment_id", { length: 256 }),

    domain: varchar("domain", { length: 256 }).notNull(),
    type: mysqlEnum("type", ["custom", "wildcard"]).notNull(),
    // sticky determines whether a domain should get reassigned to the latest deployment
    // - branch: the domain always points to the latest deployment on the branch
    //     <projectslug>-git-<branchname>-<workspaceslug>.unkey.app
    //
    // - environment: the domain is sticky to the environment it was created on
    //     <projectslug>-<environmentslug>-<workspaceslug>.unkey.app
    //
    // - live: the domain is sticky to the live deployment it was created on
    //     api.unkey.com
    sticky: mysqlEnum("sticky", ["branch", "environment", "live"]),

    // If a domain is rolled back, it does not automatically get assigned to new deployments.
    // Instead a user must manually unblock it by promoting another deployment.
    // This prevents accidentally pushing to main and deploying faulty code again.
    isRolledBack: boolean("is_rolled_back").notNull().default(false),
    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
    deploymentIdx: index("deployment_idx").on(table.deploymentId),
    uniqueDomainIdx: uniqueIndex("unique_domain_idx").on(table.domain),
  }),
);
