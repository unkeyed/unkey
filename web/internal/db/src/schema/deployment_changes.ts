import { bigint, index, mysqlEnum, mysqlTable } from "drizzle-orm/mysql-core";
import { id } from "./util/id";

export const deploymentChanges = mysqlTable(
  "deployment_changes",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    resourceType: mysqlEnum("resource_type", [
      "deployment_topology",
      "sentinel",
      "cilium_network_policy",
    ]).notNull(),
    resourceId: id("resource_id").notNull(),
    regionId: id("region_id").notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
  },
  (table) => [
    index("idx_region_type_pk").on(table.regionId, table.resourceType, table.pk),
    index("idx_created_at").on(table.createdAt),
    index("idx_region_pk").on(table.regionId, table.pk),
  ],
);
