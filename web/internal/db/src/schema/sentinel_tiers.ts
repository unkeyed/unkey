import { bigint, decimal, int, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";

/**
 * Catalog of sentinel pricing tiers. Each row represents a `(tier_id, version)`
 * pair - we insert a new `version` when pricing changes instead of mutating
 * existing rows, so subscriptions frozen against an old version keep seeing
 * the original prices. `effective_until` marks a version as retired (null
 * while currently offered).
 */
export const sentinelTiers = mysqlTable(
  "sentinel_tiers",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    tierId: varchar("tier_id", { length: 64 }).notNull(),
    version: varchar("version", { length: 32 }).notNull(),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),
    pricePerSecond: decimal("price_per_second", { precision: 12, scale: 8 }).notNull(),
    effectiveFrom: bigint("effective_from", { mode: "number" }).notNull(),
    effectiveUntil: bigint("effective_until", { mode: "number" }),
  },
  (table) => [uniqueIndex("sentinel_tiers_tier_version_unique").on(table.tierId, table.version)],
);
