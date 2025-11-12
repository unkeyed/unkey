import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { longblob } from "./util/longblob";
import { workspaces } from "./workspaces";

export const gateways = mysqlTable(
  "gateways",
  {
    id: bigint("id", { mode: "number", unsigned: true }).primaryKey().autoincrement(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    hostname: varchar("hostname", { length: 255 }).notNull(),
    config: longblob("config").notNull(), // Protobuf with all configuration including deployment_id, workspace_id
  },
  (table) => ({
    gatewaysPk: uniqueIndex("gateways_pk").on(table.hostname),
    deploymentIdIdx: index("idx_deployment_id").on(table.deploymentId),
  }),
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
