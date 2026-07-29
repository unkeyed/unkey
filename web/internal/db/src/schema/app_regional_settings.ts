import { relations } from "drizzle-orm";
import { bigint, index, int, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { environments } from "./environments";
import { horizontalAutoscalingPolicies } from "./horizontal_autoscaling_policies";
import { regions } from "./regions";
import { caseInsensitiveVarchar } from "./util/case_insensitive_varchar";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * One row per region an app+environment should be deployed to.
 * Replaces the `region_config` JSON column on `app_runtime_settings`.
 *
 * Presence of a row means "deploy this app/env to this region".
 */
export const appRegionalSettings = mysqlTable(
  "app_regional_settings",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    workspaceId: caseInsensitiveVarchar("workspace_id", { length: 256 }).notNull(),
    appId: caseSensitiveVarchar("app_id", { length: 64 }).notNull(),
    environmentId: caseSensitiveVarchar("environment_id", { length: 128 }).notNull(),
    regionId: caseSensitiveVarchar("region_id", { length: 64 }).notNull(),

    replicas: int("replicas").notNull().default(1),

    // Optional reference to a horizontal autoscaling policy. null = no autoscaling.
    horizontalAutoscalingPolicyId: caseSensitiveVarchar("horizontal_autoscaling_policy_id", {
      length: 64,
    }),

    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("unique_app_env_region").on(table.appId, table.environmentId, table.regionId),
    index("workspace_idx").on(table.workspaceId),
  ],
);

export const appRegionalSettingsRelations = relations(appRegionalSettings, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [appRegionalSettings.workspaceId],
    references: [workspaces.id],
  }),
  app: one(apps, {
    fields: [appRegionalSettings.appId],
    references: [apps.id],
  }),
  environment: one(environments, {
    fields: [appRegionalSettings.environmentId],
    references: [environments.id],
  }),
  region: one(regions, {
    fields: [appRegionalSettings.regionId],
    references: [regions.id],
  }),
  horizontalAutoscalingPolicy: one(horizontalAutoscalingPolicies, {
    fields: [appRegionalSettings.horizontalAutoscalingPolicyId],
    references: [horizontalAutoscalingPolicies.id],
  }),
}));
