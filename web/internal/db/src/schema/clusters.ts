import { relations } from "drizzle-orm";
import { bigint, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { clusterRegions } from "./cluster_regions";

// clusters tracks our kubernetes clusters
// each krane instance will heartbeat against the control plane, which then writes to this table
//
// we might use this as service discovery later to push updates to clusters to speed up reconciliation
export const clusters = mysqlTable("clusters", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

  id: varchar("id", { length: 64 }).notNull().unique(),
  regionId: varchar("region_id", { length: 64 }).notNull(),

  lastHeartbeatAt: bigint("last_heartbeat_at", {
    mode: "number",
    unsigned: true,
  }).notNull(),
});

export const regionsRelations = relations(clusters, ({ one }) => ({
  workspace: one(clusterRegions, {
    fields: [clusters.regionId],
    references: [clusterRegions.id],
  }),
}));
