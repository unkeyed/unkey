import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { workspaces } from "./workspaces";

export const gateways = mysqlTable(
  "gateways",
  {
    id: varchar("id", { length: 128 }).primaryKey(),

    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    k8sServiceName: varchar("k8s_service_name", { length: 255 }).notNull(),
    /*
     * `us-east-1`, `us-west-2` etc
     */
    region: varchar("region", { length: 255 }).notNull(),
    image: varchar("image", { length: 255 }).notNull(),
    ingressId: bigint("ingress_id", {
      mode: "number",
      unsigned: true,
    }).notNull(),
    health: mysqlEnum("health", ["paused", "healthy", "unhealthy"]), // needs better status types
  },
  (table) => [index("idx_ingress_id").on(table.ingressId)],
);

export const gatewaysRelations = relations(gateways, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [gateways.workspaceId],
    references: [workspaces.id],
  }),
  deployment: one(deployments, {
    fields: [gateways.deploymentId],
    references: [deployments.id],
  }),
}));
