import { relations } from "drizzle-orm";
import { index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { environments } from "./environments";
import { caseInsensitiveVarchar } from "./util/case_insensitive_varchar";
import { legacyId } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const ciliumNetworkPolicies = mysqlTable(
  "cilium_network_policies",
  {
    pk: primaryKey(),
    id: legacyId("id").notNull().unique(),
    workspaceId: legacyId("workspace_id").notNull(),
    projectId: legacyId("project_id").notNull(),
    appId: legacyId("app_id").notNull(),
    environmentId: legacyId("environment_id").notNull(),
    deploymentId: legacyId("deployment_id").notNull(),
    k8sName: caseInsensitiveVarchar("k8s_name", { length: 64 }).notNull(),
    k8sNamespace: varchar("k8s_namespace", { length: 255 }).notNull(),
    regionId: legacyId("region_id").notNull(),

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
