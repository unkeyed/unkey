import { relations } from "drizzle-orm";
import {
  bigint,
  boolean,
  index,
  int,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { verificationStatus } from "./custom_domains";
import { portals } from "./portals";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

/**
 * Hostnames a portal tenant attaches to its portal. Kept apart from
 * custom_domains because a portal domain routes to a portal rather than to an
 * app environment. A hostname is unique per workspace, not globally, so a
 * tenant cannot block another from claiming a domain before proving ownership.
 */
export const portalDomains = mysqlTable(
  "portal_domains",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    portalId: id("portal_id").notNull(),

    domain: varchar("domain", { length: 256 }).notNull(),

    verificationStatus: verificationStatus.notNull().default("pending"),
    verificationToken: caseSensitiveVarchar("verification_token", { length: 64 }).notNull(),
    ownershipVerified: boolean("ownership_verified").notNull().default(false),
    cnameVerified: boolean("cname_verified").notNull().default(false),
    targetCname: varchar("target_cname", { length: 256 }).notNull(),
    lastCheckedAt: bigint("last_checked_at", { mode: "number" }),
    checkAttempts: int("check_attempts").notNull().default(0),
    verificationError: varchar("verification_error", { length: 512 }),
    domainConnectProvider: varchar("domain_connect_provider", { length: 256 }),
    domainConnectUrl: varchar("domain_connect_url", { length: 2048 }),
    invocationId: varchar("invocation_id", { length: 256 }),

    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("unique_workspace_domain_idx").on(table.workspaceId, table.domain),
    uniqueIndex("unique_target_cname_idx").on(table.targetCname),
    index("domain_idx").on(table.domain),
    index("portal_id_idx").on(table.portalId),
  ],
);

export const portalDomainsRelations = relations(portalDomains, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [portalDomains.workspaceId],
    references: [workspaces.id],
  }),
  portal: one(portals, {
    fields: [portalDomains.portalId],
    references: [portals.id],
  }),
}));
