import { relations } from "drizzle-orm";
import { bigint, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { environments } from "./environments";
import { caseInsensitiveVarchar } from "./util/case_insensitive_varchar";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const ciliumNetworkPolicies = mysqlTable(
  "cilium_network_policies",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: caseSensitiveVarchar("id", { length: 32 }).notNull().unique(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 32 }).notNull(),
    projectId: caseSensitiveVarchar("project_id", { length: 32 }).notNull(),
    appId: caseSensitiveVarchar("app_id", { length: 32 }).notNull(),
    environmentId: caseSensitiveVarchar("environment_id", { length: 32 }).notNull(),
    deploymentId: caseSensitiveVarchar("deployment_id", { length: 32 }).notNull(),
    k8sName: caseInsensitiveVarchar("k8s_name", { length: 64 }).notNull(),
    k8sNamespace: varchar("k8s_namespace", { length: 255 }).notNull(),
    regionId: caseSensitiveVarchar("region_id", { length: 32 }).notNull(),

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
