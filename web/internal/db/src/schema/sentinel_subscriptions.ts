import { relations, sql } from "drizzle-orm";
import {
  bigint,
  decimal,
  index,
  int,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { sentinelTiers } from "./sentinel_tiers";

/**
 * A sentinel_subscriptions row records one config era for a single sentinel:
 * the tier, replica count, and denormalized price active from `created_at`
 * until `terminated_at`. Rows are immutable; any change to tier OR replica
 * count closes the current row (sets `terminated_at`) and inserts a new one.
 *
 * `workspace_id` and `region_id` are denormalized so aggregate queries are
 * self-sufficient — they don't need to join through `sentinels`, which is
 * important because sentinel rows are hard-deleted on env deletion while
 * subscription rows live on. The frozen copies of `cpu_millicores` /
 * `memory_mib` / `price_per_second` keep historical rows deterministic
 * after the tier catalog changes.
 *
 * The MySQL schema enforces at most one row per `sentinel_id` with
 * `terminated_at IS NULL` via a virtual-column + unique-index trick; not
 * modelled here because no TS code reads the virtual column.
 */
export const sentinelSubscriptions = mysqlTable(
  "sentinel_subscriptions",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    sentinelId: varchar("sentinel_id", { length: 64 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    regionId: varchar("region_id", { length: 255 }).notNull(),
    tierId: varchar("tier_id", { length: 64 }).notNull(),
    tierVersion: varchar("tier_version", { length: 32 }).notNull(),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),
    replicas: int("replicas").notNull(),
    pricePerSecond: decimal("price_per_second", { precision: 12, scale: 8 }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    terminatedAt: bigint("terminated_at", { mode: "number" }),
    // Virtual column that holds sentinel_id only while the subscription is
    // open. Combined with a unique index below, this enforces "at most one
    // open subscription per sentinel" at the DB level. MySQL's NULL-is-not-
    // equal semantics lets every terminated row coexist.
    openSentinelId: varchar("open_sentinel_id", { length: 64 }).generatedAlwaysAs(
      sql`(CASE WHEN \`terminated_at\` IS NULL THEN \`sentinel_id\` ELSE NULL END)`,
      { mode: "virtual" },
    ),
  },
  (table) => [
    index("idx_sentinel_created").on(table.sentinelId, table.createdAt),
    index("idx_workspace_period").on(table.workspaceId, table.createdAt, table.terminatedAt),
    uniqueIndex("one_open_subscription_per_sentinel").on(table.openSentinelId),
  ],
);

export const sentinelSubscriptionsRelations = relations(sentinelSubscriptions, ({ one }) => ({
  tier: one(sentinelTiers, {
    fields: [sentinelSubscriptions.tierId, sentinelSubscriptions.tierVersion],
    references: [sentinelTiers.tierId, sentinelTiers.version],
  }),
}));
