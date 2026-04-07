import { relations } from "drizzle-orm";
import { bigint, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { environments } from "./environments";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const ciliumNetworkPolicies = mysqlTable(
  "cilium_network_policies",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 128 }).notNull(),
    k8sName: varchar("k8s_name", { length: 64 }).notNull(),
    k8sNamespace: varchar("k8s_namespace", { length: 255 }).notNull(),
    regionId: varchar("region_id", { length: 64 }).notNull(),

    // json representation of the policy
    policy: json("policy").notNull(),

    ...lifecycleDates,
  },
  (table) => [index("idx_deployment_region").on(table.deploymentId, table.regionId)],
);

export const ciliumNetworkPoliciesRelations = relations(ciliumNetworkPolicies, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [ciliumNetworkPolicies.workspaceId],
    references: [workspaces.id],
  }),
  environment: one(environments, {
    fields: [ciliumNetworkPolicies.environmentId],
    references: [environments.id],
  }),
  deployment: one(deployments, {
    fields: [ciliumNetworkPolicies.deploymentId],
    references: [deployments.id],
  }),
}));
