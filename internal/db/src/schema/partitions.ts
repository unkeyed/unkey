import { relations } from "drizzle-orm";
import { index, mysqlEnum, mysqlTable, text, varchar } from "drizzle-orm/mysql-core";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";

export const partitions = mysqlTable(
  "partitions",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    name: varchar("name", { length: 256 }).notNull(),
    description: text("description"),

    // AWS account information
    awsAccountId: varchar("aws_account_id", { length: 256 }).notNull(),
    region: varchar("region", { length: 256 }).notNull(), // Primary AWS region

    // Network configuration
    ipV4Address: varchar("ip_v4_address", { length: 15 }),
    ipV6Address: varchar("ip_v6_address", { length: 39 }),

    // Status management
    status: mysqlEnum("status", ["active", "draining", "inactive"]).notNull().default("active"),

    ...deleteProtection,
    ...lifecycleDatesMigration,
  },
  (table) => ({
    statusIdx: index("status_idx").on(table.status),
  }),
);

export const partitionsRelations = relations(partitions, () => ({
  // workspaces: many(workspaces), // We'll add this after updating workspaces table
  // metalHosts: many(metalHosts),
  // regions: many(regions),
}));
