import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable } from "drizzle-orm/mysql-core";
import { customDomains } from "./custom_domains";
import { caseInsensitiveVarchar } from "./util/case_insensitive_varchar";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const challengeType = mysqlEnum("challenge_type", ["HTTP-01", "DNS-01"]).notNull();

export const acmeChallenges = mysqlTable(
  "acme_challenges",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    domainId: caseSensitiveVarchar("domain_id", { length: 255 }).notNull().unique(),
    workspaceId: caseInsensitiveVarchar("workspace_id", { length: 255 }).notNull(),
    token: caseInsensitiveVarchar("token", { length: 255 }).notNull(),
    type: challengeType,
    authorization: caseInsensitiveVarchar("authorization", { length: 255 }).notNull(),
    status: mysqlEnum("status", ["waiting", "pending", "verified", "failed"]).notNull(),
    expiresAt: bigint("expires_at", { mode: "number" }).notNull(),

    ...lifecycleDates,
  },
  (table) => [index("workspace_idx").on(table.workspaceId), index("status_idx").on(table.status)],
);

export const acmeChallengeRelations = relations(acmeChallenges, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [acmeChallenges.workspaceId],
    references: [workspaces.id],
  }),
  domain: one(customDomains, {
    fields: [acmeChallenges.domainId],
    references: [customDomains.id],
  }),
}));
