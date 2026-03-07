import { relations } from "drizzle-orm";
import {
  bigint,
  index,
  int,
  mysqlTable,
  tinyint,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { environments } from "./environments";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * One row per region an app+environment should be deployed to.
 * Replaces the `region_config` JSON column on `app_runtime_settings`.
 *
 * Autoscaling is considered enabled if replicasMax > replicasMin and there is at least one threshold configured (memoryThreshold, cpuThreshold, or rpsThreshold).
 * If autoscaling is enabled, the system will scale up the app when any of the specified thresholds are breached, and scale down when they are no longer breached,
 * within the limits of replicasMin and replicasMax.
 */
export const appScalingSettings = mysqlTable(
  "app_scaling_settings",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),
    regionId: varchar("region_id", { length: 64 }).notNull(),

    // 0-100, the percentage threshold that triggers scaling
    memoryThreshold: tinyint("memory_threshold"),
    cpuThreshold: tinyint("cpu_threshold"),
    rpsThreshold: tinyint("rps_threshold"),

    // set both replicasMin and replicasMax to the same value to disable autoscaling
    replicasMin: int("replicas_min").notNull(),
    replicasMax: int("replicas_max").notNull(),

    cpuMillicores: int("max_cpu_millicores").notNull(),
    memoryMib: int("max_memory_mib").notNull(),

    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("unique_app_env_region").on(table.appId, table.environmentId, table.regionId),
    index("workspace_idx").on(table.workspaceId),
  ],
);

export const appScalingSettingsRelations = relations(appScalingSettings, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [appScalingSettings.workspaceId],
    references: [workspaces.id],
  }),
  app: one(apps, {
    fields: [appScalingSettings.appId],
    references: [apps.id],
  }),
  environment: one(environments, {
    fields: [appScalingSettings.environmentId],
    references: [environments.id],
  }),
}));
