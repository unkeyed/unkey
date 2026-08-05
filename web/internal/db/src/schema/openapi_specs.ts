import { relations } from "drizzle-orm";
import { bigint, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";
import { workspaces } from "./workspaces";

export const openapiSpecs = mysqlTable(
  "openapi_specs",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    deploymentId: id("deployment_id"),
    portalConfigId: id("portal_config_id"),
    content: longblob("content").notNull(),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("idx_openapi_specs_on_deployment_id").on(table.deploymentId),
    uniqueIndex("workspace_deployment_idx").on(table.workspaceId, table.deploymentId),
    uniqueIndex("workspace_portal_config_idx").on(table.workspaceId, table.portalConfigId),
  ],
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
