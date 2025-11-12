import { relations } from "drizzle-orm";
import { index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { longblob } from "./util/longblob";

export const instances = mysqlTable(
  "instances",
  {
    id: varchar("id", { length: 255 }).primaryKey(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    status: mysqlEnum("status", [
      "allocated",
      "provisioning",
      "starting",
      "running",
      "stopping",
      "stopped",
      "failed",
    ]).notNull(),
    config: longblob("config").notNull(),
  },
  (table) => ({
    deploymentIdIdx: index("idx_deployment_id").on(table.deploymentId),
  })
);

export const vmsRelations = relations(vms, ({ one }) => ({
  deployment: one(deployments, {
    fields: [instances.deploymentId],
    references: [deployments.id],
  }),
}));
