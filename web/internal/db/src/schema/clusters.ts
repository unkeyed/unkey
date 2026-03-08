import { relations } from "drizzle-orm";
import { bigint, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { regions } from "./regions";

// clusters tracks our kubernetes clusters
// each krane instance will heartbeat against the control plane, which then writes to this table
//
// we might use this as service discovery later to push updates to clusters to speed up reconciliation
export const clusters = mysqlTable("clusters", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

  id: varchar("id", { length: 64 }).notNull().unique(),
  regionId: varchar("region_id", { length: 64 }).notNull().unique(),

  lastHeartbeatAt: bigint("last_heartbeat_at", {
    mode: "number",
    unsigned: true,
  }).notNull(),
});

export const clustersRelations = relations(clusters, ({ one }) => ({
  region: one(regions, {
    fields: [clusters.regionId],
    references: [regions.id],
  }),
}));
