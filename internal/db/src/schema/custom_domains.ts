import { bigint, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { challengeType } from "./acme_challenges";
import { lifecycleDates } from "./util/lifecycle_dates";

export const customDomains = mysqlTable(
  "custom_domains",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 128 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    domain: varchar("domain", { length: 256 }).notNull(),
    challengeType: challengeType,

    ...lifecycleDates,
  },
  (table) => [
    index("workspace_idx").on(table.workspaceId),
    uniqueIndex("unique_domain_idx").on(table.domain),
  ],
);
