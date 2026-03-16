import { bigint, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";

export const openapiSpecs = mysqlTable(
  "openapi_specs",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 64 }),
    // null = user-uploaded spec not linked to a specific deployment
    deploymentId: varchar("deployment_id", { length: 64 }),
    spec: longblob("spec").notNull(),
    ...lifecycleDates,
  },
  (table) => [
    // each deployment may have at most one spec; MySQL allows multiple NULLs in a unique index
    uniqueIndex("openapi_specs_deployment_idx").on(table.deploymentId),
    index("openapi_specs_project_idx").on(table.projectId),
  ],
);
