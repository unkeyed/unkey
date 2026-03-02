import { relations } from "drizzle-orm";
import { bigint, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { clusters } from "./clusters";

export const regions = mysqlTable(
  "regions",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    name: varchar("name", { length: 64 }).unique().notNull(),
    platform: mysqlEnum("platform", ["aws", "local"]).notNull(),
  },
  (table) => [uniqueIndex("unique_reqion_per_platform").on(table.name, table.platform)],
);

export const regionsRelations = relations(regions, ({ many }) => ({
  clusters: many(clusters),
}));
