import { index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
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
    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
  }),
);
