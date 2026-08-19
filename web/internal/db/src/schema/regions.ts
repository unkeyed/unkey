import { relations } from "drizzle-orm";
import { boolean, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { clusters } from "./clusters";
import { caseInsensitiveVarchar } from "./util/case_insensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const regions = mysqlTable(
  "regions",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    // e.g. us-east-1, us-west-2, etc.
    name: caseInsensitiveVarchar("name", { length: 64 }).notNull(),
    // e.g. aws, gcp, azure, local, etc.
    platform: caseInsensitiveVarchar("platform", { length: 64 }).notNull(),
    // Whether this region is available for users to schedule deployments to.
    // Defaults to true — set to false to hide a region from scheduling.
    canSchedule: boolean("can_schedule").notNull().default(true),
  },
  (table) => [uniqueIndex("unique_region_per_platform").on(table.name, table.platform)],
);

export const regionRelations = relations(regions, ({ many }) => ({
  clusters: many(clusters),
}));
