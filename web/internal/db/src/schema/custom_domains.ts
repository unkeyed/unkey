import {
  bigint,
  boolean,
  index,
  int,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
} from "drizzle-orm/mysql-core";
import { challengeType } from "./acme_challenges";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
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
    id: caseSensitiveVarchar("id", { length: 128 }).notNull().unique(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 256 }).notNull(),
    projectId: caseSensitiveVarchar("project_id", { length: 256 }).notNull(),
    appId: caseSensitiveVarchar("app_id", { length: 64 }).notNull(),
    environmentId: caseSensitiveVarchar("environment_id", { length: 256 }).notNull(),

    domain: caseSensitiveVarchar("domain", { length: 256 }).notNull(),
    challengeType: challengeType,

    // Verification fields
    verificationStatus: verificationStatus.notNull().default("pending"),
    // TXT record verification token (e.g., "abc123xyz...")
    // User adds TXT record: _unkey.domain.com -> unkey-domain-verify=<token>
    verificationToken: caseSensitiveVarchar("verification_token", { length: 64 }).notNull(),
    // Whether the TXT record has been verified (proves ownership)
    ownershipVerified: boolean("ownership_verified").notNull().default(false),
    // Whether the CNAME record has been verified (enables routing)
    cnameVerified: boolean("cname_verified").notNull().default(false),
    // Unique CNAME target for this domain (e.g., "k3n5p8x2")
    // Combined with base domain to form full target like "k3n5p8x2.cname.unkey.local"
    targetCname: caseSensitiveVarchar("target_cname", { length: 256 }).notNull().unique(),
    lastCheckedAt: bigint("last_checked_at", { mode: "number" }),
    checkAttempts: int("check_attempts").notNull().default(0),
    verificationError: caseSensitiveVarchar("verification_error", { length: 512 }),
    domainConnectProvider: caseSensitiveVarchar("domain_connect_provider", { length: 256 }),
    domainConnectUrl: caseSensitiveVarchar("domain_connect_url", { length: 2048 }),
    invocationId: caseSensitiveVarchar("invocation_id", { length: 256 }),

    ...lifecycleDates,
  },
  (table) => [
    index("project_idx").on(table.projectId),
    index("verification_status_idx").on(table.verificationStatus),
    uniqueIndex("unique_domain_workspace_idx").on(table.workspaceId, table.domain),
  ],
);
