import { relations } from "drizzle-orm";
import { bigint, json, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
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
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 128 }).notNull(),
    k8sName: varchar("k8s_name", { length: 64 }).notNull(),
    k8sNamespace: varchar("k8s_namespace", { length: 255 }).notNull(),
    region: varchar("region", { length: 255 }).notNull(),

    // json representation of the policy
    policy: json("policy").notNull(),

    // Version for state synchronization with edge agents.
    // Updated via Restate VersioningService on each mutation.
    // Edge agents track their last-seen version and request changes after it.
    // Unique per region (composite index with region).
    version: bigint("version", { mode: "number", unsigned: true }).notNull(),

    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("one_deployment_per_region").on(table.deploymentId, table.region, table.k8sName),
    uniqueIndex("unique_version_per_region").on(table.region, table.version),
  ],
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
