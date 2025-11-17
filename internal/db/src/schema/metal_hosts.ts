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
import { vms } from "./vms";

export const metalHosts = mysqlTable(
  "metal_hosts",
  {
    id: varchar("id", { length: 255 }).primaryKey(),
    region: varchar("region", { length: 255 }).notNull(),
    availabilityZone: varchar("availability_zone", { length: 255 }).notNull(),
    instanceType: varchar("instance_type", { length: 255 }).notNull(),
    ec2InstanceId: varchar("ec2_instance_id", { length: 255 }).notNull(),
    privateIp: varchar("private_ip", { length: 45 }).notNull(),
    status: mysqlEnum("status", ["provisioning", "active", "draining", "terminated"]).notNull(),
    capacityCpuMillicores: int("capacity_cpu_millicores").notNull(),
    capacityMemoryMb: int("capacity_memory_mb").notNull(),
    allocatedCpuMillicores: int("allocated_cpu_millicores").notNull().default(0),
    allocatedMemoryMb: int("allocated_memory_mb").notNull().default(0),
    lastHeartbeat: bigint("last_heartbeat", { mode: "number" }).notNull(),
  },
  (table) => ({
    regionStatusIdx: index("idx_region_status").on(table.region, table.status),
    azIdx: index("idx_az").on(table.availabilityZone),
    statusIdx: index("idx_status").on(table.status),
    heartbeatIdx: index("idx_heartbeat").on(table.lastHeartbeat),
    uniqueEc2Instance: uniqueIndex("unique_ec2_instance").on(table.ec2InstanceId),
  }),
);

export const metalHostsRelations = relations(metalHosts, ({ many }) => ({
  vms: many(vms),
}));
