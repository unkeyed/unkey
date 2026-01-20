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
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * We store one row per logical sentinel. That means each set of sentinel pods in a single region is one row.
 * Therefore each sentinel also has a single kubernetes service name.
 */
export const sentinels = mysqlTable(
  "sentinels",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    k8sName: varchar("k8s_name", { length: 64 }).notNull().unique(),
    k8sAddress: varchar("k8s_address", { length: 255 }).notNull().unique(),
    /*
     * `us-east-1`, `us-west-2` etc
     */
    region: varchar("region", { length: 255 }).notNull(),
    image: varchar("image", { length: 255 }).notNull(),
    desiredState: mysqlEnum("desired_state", ["running", "standby", "archived"])
      .notNull()
      .default("running"),

    health: mysqlEnum("health", ["unknown", "paused", "healthy", "unhealthy"])
      .notNull()
      .default("unknown"), // needs better status types
    desiredReplicas: int("desired_replicas").notNull(),
    availableReplicas: int("available_replicas").notNull(),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),

    // Version for state synchronization with edge agents.
    // Updated via Restate VersioningService on each mutation.
    // Edge agents track their last-seen version and request changes after it.
    // Unique across all resources (shared global counter).
    version: bigint("version", { mode: "number", unsigned: true }).notNull().unique(),

    ...lifecycleDates,
  },
  (table) => [
    index("idx_environment_id").on(table.environmentId),
    index("region_version_idx").on(table.region, table.version),
    uniqueIndex("one_env_per_region").on(table.environmentId, table.region),
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
}));
