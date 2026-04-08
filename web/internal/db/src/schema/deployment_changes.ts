import { bigint, index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";

export const deploymentChanges = mysqlTable(
  "deployment_changes",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    resourceType: mysqlEnum("resource_type", [
      "deployment_topology",
      "sentinel",
      "cilium_network_policy",
    ]).notNull(),
    resourceId: varchar("resource_id", { length: 64 }).notNull(),
    regionId: varchar("region_id", { length: 64 }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
  },
  (table) => [
    index("idx_region_type_pk").on(table.regionId, table.resourceType, table.pk),
    index("idx_created_at").on(table.createdAt),
    index("idx_region_pk").on(table.regionId, table.pk),
  ],
);
