import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { customDomains } from "./custom_domains";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const challengeType = mysqlEnum("challenge_type", ["HTTP-01", "DNS-01"]).notNull();

export const acmeChallenges = mysqlTable(
  "acme_challenges",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    domainId: varchar("domain_id", { length: 255 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    token: varchar("token", { length: 255 }).notNull(),
    type: challengeType,
    authorization: varchar("authorization", { length: 255 }).notNull(),
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
