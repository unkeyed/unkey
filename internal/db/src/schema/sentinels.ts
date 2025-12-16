import { relations } from "drizzle-orm";
import {
  index,
  int,
  mysqlEnum,
  mysqlTable,
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
    id: varchar("id", { length: 128 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    k8sCrdName: varchar("k8s_crd_name", { length: 255 }).notNull().unique(),
    k8sServiceName: varchar("k8s_service_name", { length: 255 })
      .notNull()
      .unique(),
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
    replicas: int("replicas").notNull(),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),
    ...lifecycleDates,
  },
  (table) => [index("idx_environment_id").on(table.environmentId)]
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
