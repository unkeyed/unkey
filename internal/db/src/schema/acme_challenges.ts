import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { domains } from "./domains";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const acmeChallenges = mysqlTable(
  "acme_challenges",
  {
    id: bigint("id", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    domainId: varchar("domain_id", { length: 256 }).notNull(),
    token: varchar("token", { length: 256 }).notNull(),
    authorization: varchar("authorization", { length: 256 }).notNull(),
    status: mysqlEnum("status", ["waiting", "pending", "verified", "failed"]).notNull(),
    expiresAt: bigint("expires_at", { mode: "number" }).notNull(),

    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
  }),
);

export const acmeChallengeRelations = relations(acmeChallenges, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [acmeChallenges.workspaceId],
    references: [workspaces.id],
  }),
  domain: one(domains, {
    fields: [acmeChallenges.domainId],
    references: [domains.id],
  }),
}));
