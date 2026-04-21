import { relations } from "drizzle-orm";
import {
  bigint,
  index,
  int,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { environments } from "./environments";
import { regions } from "./regions";
import { sentinelSubscriptions } from "./sentinel_subscriptions";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * We store one row per logical sentinel. That means each set of sentinel pods in a single region is one row.
 * Therefore each sentinel also has a single kubernetes service name.
 *
 * `image` is the desired image (what we want deployed).
 * `running_image` is the observed image (what krane reports k8s is actually running).
 * When they differ, the last deploy hasn't converged yet.
 */
export const sentinels = mysqlTable(
  "sentinels",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    subscriptionId: varchar("subscription_id", { length: 64 }).notNull(),
    k8sName: varchar("k8s_name", { length: 64 }).notNull().unique(),
    k8sAddress: varchar("k8s_address", { length: 255 }).notNull().unique(),

    regionId: varchar("region_id", { length: 255 }).notNull(),
    image: varchar("image", { length: 255 }).notNull(),
    runningImage: varchar("running_image", { length: 255 }).notNull().default(""),
    desiredState: mysqlEnum("desired_state", ["running", "standby", "archived"])
      .notNull()
      .default("running"),

    health: mysqlEnum("health", ["unknown", "paused", "healthy", "unhealthy"])
      .notNull()
      .default("unknown"),
    desiredReplicas: int("desired_replicas").notNull(),
    availableReplicas: int("available_replicas").notNull().default(0),
    deployStatus: mysqlEnum("deploy_status", ["idle", "progressing", "ready", "failed"])
      .notNull()
      .default("idle"),

    ...lifecycleDates,
  },
  (table) => [
    index("idx_environment_health_region_routing").on(
      table.environmentId,
      table.regionId,
      table.health,
    ),
    uniqueIndex("one_env_per_region").on(table.environmentId, table.regionId),
  ],
);

export const sentinelsRelations = relations(sentinels, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [sentinels.workspaceId],
    references: [workspaces.id],
  }),
  environment: one(environments, {
    fields: [sentinels.environmentId],
    references: [environments.id],
  }),
  region: one(regions, {
    fields: [sentinels.regionId],
    references: [regions.id],
  }),
  subscription: one(sentinelSubscriptions, {
    fields: [sentinels.subscriptionId],
    references: [sentinelSubscriptions.id],
  }),
}));
