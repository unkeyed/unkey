import { relations } from "drizzle-orm";
import { bigint, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";
import { workspaces } from "./workspaces";

export const openapiSpecs = mysqlTable(
  "openapi_specs",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 128 }),
    portalConfigId: varchar("portal_config_id", { length: 256 }),

    spec: longblob("spec").notNull(),

    ...lifecycleDates,
  },
  (table) => [uniqueIndex("openapi_specs_deployment_id_unique").on(table.deploymentId)],
);

export const openapiSpecsRelations = relations(openapiSpecs, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [openapiSpecs.workspaceId],
    references: [workspaces.id],
  }),
  deployment: one(deployments, {
    fields: [openapiSpecs.deploymentId],
    references: [deployments.id],
  }),
}));
