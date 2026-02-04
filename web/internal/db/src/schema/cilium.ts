import { relations } from "drizzle-orm";
import { bigint, index, json, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
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
    k8sName: varchar("k8s_name", { length: 64 }).notNull().unique(),
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
    index("idx_environment_id").on(table.environmentId),
    uniqueIndex("one_env_per_region").on(table.environmentId, table.region),
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
}));
