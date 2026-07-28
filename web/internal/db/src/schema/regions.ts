import { relations } from "drizzle-orm";
import { bigint, boolean, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { clusters } from "./clusters";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";

export const regions = mysqlTable(
  "regions",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: caseSensitiveVarchar("id", { length: 32 }).notNull().unique(),
    // e.g. us-east-1, us-west-2, etc.
    name: caseSensitiveVarchar("name", { length: 64 }).notNull(),
    // e.g. aws, gcp, azure, local, etc.
    platform: caseSensitiveVarchar("platform", { length: 64 }).notNull(),
    // Whether this region is available for users to schedule deployments to.
    // Defaults to true — set to false to hide a region from scheduling.
    canSchedule: boolean("can_schedule").notNull().default(true),
  },
  (table) => [uniqueIndex("unique_region_per_platform").on(table.name, table.platform)],
);

export const regionRelations = relations(regions, ({ many }) => ({
  clusters: many(clusters),
}));
