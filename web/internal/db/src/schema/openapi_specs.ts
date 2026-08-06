import { relations } from "drizzle-orm";
import { mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { legacyId } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const openapiSpecs = mysqlTable(
  "openapi_specs",
  {
    pk: primaryKey(),
    id: legacyId("id").notNull().unique(),
    workspaceId: legacyId("workspace_id").notNull(),
    deploymentId: legacyId("deployment_id"),
    portalConfigId: legacyId("portal_config_id"),
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
