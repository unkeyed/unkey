import { relations } from "drizzle-orm";
import { bigint, mysqlTable } from "drizzle-orm/mysql-core";
import { regions } from "./regions";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
// import { id } from "./util/id";

// clusters tracks our kubernetes clusters
// each krane instance will heartbeat against the control plane, which then writes to this table
//
// we might use this as service discovery later to push updates to clusters to speed up reconciliation
export const clusters = mysqlTable("clusters", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

  // id: id("id").notNull().unique(),
  id: caseSensitiveVarchar("id", { length: 64 }).notNull().unique(),
  // Nullable until an existing cluster reports its cell identity by heartbeat.
  cellId: caseSensitiveVarchar("cell_id", { length: 64 }).unique(),
  // regionId: id("region_id").notNull().unique(),
  regionId: caseSensitiveVarchar("region_id", { length: 64 }).notNull().unique(),

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
