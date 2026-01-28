import {
  bigint,
  index,
  int,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { challengeType } from "./acme_challenges";
import { lifecycleDates } from "./util/lifecycle_dates";

export const verificationStatus = mysqlEnum("verification_status", [
  "pending",
  "verifying",
  "verified",
  "failed",
]);

export const customDomains = mysqlTable(
  "custom_domains",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 128 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    environmentId: varchar("environment_id", { length: 256 }).notNull(),

    domain: varchar("domain", { length: 256 }).notNull(),
    challengeType: challengeType,

    // Verification fields
    verificationStatus: verificationStatus.notNull().default("pending"),
    targetCname: varchar("target_cname", { length: 256 }).notNull(),
    lastCheckedAt: bigint("last_checked_at", { mode: "number" }),
    checkAttempts: int("check_attempts").notNull().default(0),
    verificationError: varchar("verification_error", { length: 512 }),
    invocationId: varchar("invocation_id", { length: 256 }),

    ...lifecycleDates,
  },
  (table) => [
    index("workspace_idx").on(table.workspaceId),
    index("project_idx").on(table.projectId),
    index("verification_status_idx").on(table.verificationStatus),
    uniqueIndex("unique_domain_idx").on(table.domain),
  ],
);
