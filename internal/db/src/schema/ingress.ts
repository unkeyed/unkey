import { relations } from "drizzle-orm";
import { mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { gateways } from "./gateways";
import { projects } from "./projects";
import { workspaces } from "./workspaces";

export const ingresses = mysqlTable("ingresses", {
  id: varchar("id", { length: 128 }).primaryKey(),
  hostname: varchar("hostname", { length: 255 }).notNull().unique(),
  workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
  projectId: varchar("project_id", { length: 255 }).notNull(),
  deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
});

export const ingressRelations = relations(ingresses, ({ one, many }) => ({
  deployment: one(deployments, {
    fields: [ingresses.deploymentId],
    references: [deployments.id],
  }),
  workspace: one(workspaces, {
    fields: [ingresses.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [ingresses.projectId],
    references: [projects.id],
  }),
  gateways: many(gateways),
}));
