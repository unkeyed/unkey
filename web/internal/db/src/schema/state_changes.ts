import { bigint, index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";

/**
 * stateChanges is a lightweight changelog for Kubernetes-style List+Watch sync.
 *
 * When cluster resources (deployments, sentinels) are created, updated, or deleted,
 * a row is inserted here. Cluster agents (krane) poll this table to receive incremental
 * updates instead of re-fetching all resources on every sync.
 *
 * The sequence column provides a monotonically increasing watermark. Agents persist their
 * last-seen sequence and resume from there on reconnect, avoiding full resyncs.
 *
 * For upserts, the actual resource config is fetched via existing queries.
 * For deletes, the resource row (deployment/sentinel) is soft-deleted so the k8s
 * identity can still be looked up.
 *
 * Retention: Rows older than 7 days are periodically deleted. Agents with a watermark
 * behind the minimum retained sequence must perform a full resync.
 */
export const stateChanges = mysqlTable(
  "state_changes",
  {
    sequence: bigint("sequence", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    resourceType: mysqlEnum("resource_type", ["sentinel", "deployment"]).notNull(),
    resourceId: varchar("resource_id", { length: 256 }).notNull(),
    op: mysqlEnum("op", ["upsert", "delete"]).notNull(),

    region: varchar("region", { length: 64 }).notNull(),

    createdAt: bigint("created_at", {
      mode: "number",
      unsigned: true,
    }).notNull(),
  },
  (table) => [
    index("region_sequence").on(table.region, table.sequence),
    index("created_at").on(table.createdAt),
  ],
);
