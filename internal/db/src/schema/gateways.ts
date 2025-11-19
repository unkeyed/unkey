import { relations } from "drizzle-orm";
import {
  index,
  int,
  mysqlEnum,
  mysqlTable,
  varchar,
} from "drizzle-orm/mysql-core";
import { environments } from "./environments";
import { workspaces } from "./workspaces";

/**
 * We store one row per logical gateway. That means each set of gateway pods in a single region is one row.
 * Therefore each gateway also has a single kubernetes service name.
 */
export const gateways = mysqlTable(
  "gateways",
  {
    id: varchar("id", { length: 128 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    environmentId: varchar("environment_id", { length: 255 }).notNull(),
    k8sServiceName: varchar("k8s_service_name", { length: 255 }).notNull(),
    /*
     * `us-east-1`, `us-west-2` etc
     */
    region: varchar("region", { length: 255 }).notNull(),
    image: varchar("image", { length: 255 }).notNull(),
    health: mysqlEnum("health", ["paused", "healthy", "unhealthy"]), // needs better status types
    replicas: int("replicas").notNull(),
  },
  (table) => [index("idx_environment_id").on(table.environmentId)]
);

export const gatewaysRelations = relations(gateways, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [gateways.workspaceId],
    references: [workspaces.id],
  }),
  environment: one(environments, {
    fields: [gateways.environmentId],
    references: [environments.id],
  }),
}));
