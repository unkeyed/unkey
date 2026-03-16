import { relations } from "drizzle-orm";
import { bigint, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { projects } from "./projects";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";
import { workspaces } from "./workspaces";

export const openapiSpecs = mysqlTable(
  "openapi_specs",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 128 }).notNull().unique(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    // null = user-uploaded spec not linked to a specific deployment
    projectId: varchar("project_id", { length: 64 }),
    deploymentId: varchar("deployment_id", { length: 128 }),

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
  project: one(projects, {
    fields: [openapiSpecs.projectId],
    references: [projects.id],
  }),
  deployment: one(deployments, {
    fields: [openapiSpecs.deploymentId],
    references: [deployments.id],
  }),
}));
