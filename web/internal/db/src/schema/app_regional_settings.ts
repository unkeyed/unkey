import { relations } from "drizzle-orm";
import { index, int, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { environments } from "./environments";
import { horizontalAutoscalingPolicies } from "./horizontal_autoscaling_policies";
import { regions } from "./regions";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
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
    pk: primaryKey(),

    workspaceId: id("workspace_id").notNull(),
    appId: id("app_id").notNull(),
    environmentId: id("environment_id").notNull(),
    regionId: id("region_id").notNull(),

    replicas: int("replicas").notNull().default(1),

    // Optional reference to a horizontal autoscaling policy. null = no autoscaling.
    horizontalAutoscalingPolicyId: id("horizontal_autoscaling_policy_id"),

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
