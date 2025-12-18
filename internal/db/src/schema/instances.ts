import { relations } from "drizzle-orm";
import {
  index,
  int,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { projects } from "./projects";

//id, deplyoment_id, health, kube_dns_addr, mem, cpu, region

export const instances = mysqlTable(
  "instances",
  {
    id: varchar("id", { length: 128 }).primaryKey(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    region: varchar("region", { length: 255 }).notNull(),
    // allows multiple clusters per region later
    shard: varchar("shard", { length: 255 }).notNull(),

    // used to apply updates from the kubernetes watch events
    k8sName: varchar("k8s_name", { length: 255 }).notNull(),
    // the kubernetes pod dns address from the stateful set
    address: varchar("address", { length: 255 }).notNull(),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),
    status: mysqlEnum("status", [
      "inactive",
      "pending",
      "running",
      "failed",
    ]).notNull(),
  },
  (table) => [
    uniqueIndex("unique_address_per_shard").on(table.address, table.shard),
    uniqueIndex("unique_pod_name_per_shard").on(table.k8sName, table.shard),
    index("idx_deployment_id").on(table.deploymentId),
    index("idx_region").on(table.region),
  ]
);

export const instancesRelations = relations(instances, ({ one }) => ({
  deployment: one(deployments, {
    fields: [instances.deploymentId],
    references: [deployments.id],
  }),
  project: one(projects, {
    fields: [instances.projectId],
    references: [projects.id],
  }),
}));
