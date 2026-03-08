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
import { deployments } from "./deployments";
import { projects } from "./projects";
import { regions } from "./regions";

//id, deplyoment_id, health, kube_dns_addr, mem, cpu, region

export const instances = mysqlTable(
  "instances",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),

    region: varchar("region", { length: 255 }).default("DELETE ME"),
    regionId: varchar("region_id", { length: 64 }).notNull().default("TODO"),

    // used to apply updates from the kubernetes watch events
    k8sName: varchar("k8s_name", { length: 255 }).notNull(),
    // the kubernetes pod dns address from the stateful set
    address: varchar("address", { length: 255 }).notNull(),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),
    status: mysqlEnum("status", ["inactive", "pending", "running", "failed"]).notNull(),
  },
  (table) => [
    uniqueIndex("unique_address_per_region").on(table.address, table.regionId),
    uniqueIndex("unique_k8s_name_per_region").on(table.k8sName, table.regionId),
    index("idx_deployment_id").on(table.deploymentId),
    index("idx_region").on(table.regionId),
  ],
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
  region: one(regions, {
    fields: [instances.regionId],
    references: [regions.id],
  }),
}));
