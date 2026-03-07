import { relations } from "drizzle-orm";
import { bigint, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { clusters } from "./clusters";

export const clusterRegions = mysqlTable(
  "cluster_regions",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    // e.g. us-east-1, us-west-2, etc.
    name: varchar("name", { length: 64 }).unique().notNull(),
    // e.g. aws, gcp, azure, local, etc.
    platform: varchar("platform", { length: 64 }).notNull(),
  },
  (table) => [uniqueIndex("unique_reqion_per_platform").on(table.name, table.platform)],
);

export const clusterRegionRelations = relations(clusterRegions, ({ many }) => ({
  clusters: many(clusters),
}));
