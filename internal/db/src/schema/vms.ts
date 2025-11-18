import { relations } from "drizzle-orm";
import { index, int, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { metalHosts } from "./metal_hosts";

export const vms = mysqlTable(
  "vms",
  {
    id: varchar("id", { length: 255 }).primaryKey(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    metalHostId: varchar("metal_host_id", { length: 255 }), // NULL until assigned to a host
    address: varchar("address", { length: 255 }), // NULL until provisioned
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMb: int("memory_mb").notNull(),
    status: mysqlEnum("status", [
      "allocated",
      "provisioning",
      "starting",
      "running",
      "stopping",
      "stopped",
      "failed",
    ]).notNull(),
  },
  (table) => ({
    uniqueAddress: uniqueIndex("unique_address").on(table.address),
    deploymentIdIdx: index("idx_deployment_id").on(table.deploymentId),
  }),
);

export const vmsRelations = relations(vms, ({ one }) => ({
  deployment: one(deployments, {
    fields: [vms.deploymentId],
    references: [deployments.id],
  }),
  metalHost: one(metalHosts, {
    fields: [vms.metalHostId],
    references: [metalHosts.id],
  }),
}));
