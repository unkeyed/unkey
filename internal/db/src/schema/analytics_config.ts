import { relations } from "drizzle-orm";
import { boolean, index, json, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const analyticsConfig = mysqlTable(
  "analytics_config",
  {
    workspaceId: varchar("workspace_id", { length: 255 }).notNull().primaryKey(),
    storage: mysqlEnum("storage", ["iceberg", "clickhouse"]).notNull(),
    enabled: boolean("enabled").notNull().default(false),
    config: json("config").notNull().default({}),
    ...lifecycleDates,
  },
  (table) => ({
    domainIdx: index("domain_idx").on(table.workspaceId),
  }),
);

export const analyticsConfigRelations = relations(analyticsConfig, ({ one }) => ({
  workspace: one(workspaces),
}));
