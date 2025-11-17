import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable, primaryKey, varchar } from "drizzle-orm/mysql-core";
import { customDomains } from "./custom_domains";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const acmeChallenges = mysqlTable(
  "acme_challenges",
  {
    domainId: varchar("domain_id", { length: 255 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    token: varchar("token", { length: 255 }).notNull(),
    type: mysqlEnum("type", ["HTTP-01", "DNS-01"]).notNull(),
    authorization: varchar("authorization", { length: 255 }).notNull(),
    status: mysqlEnum("status", ["waiting", "pending", "verified", "failed"]).notNull(),
    expiresAt: bigint("expires_at", { mode: "number" }).notNull(),

    ...lifecycleDates,
  },
  (table) => ({
    pk: primaryKey({ columns: [table.domainId] }),
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    statusIdx: index("status_idx").on(table.status),
  }),
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
