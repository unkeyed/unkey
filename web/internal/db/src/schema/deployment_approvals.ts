import { relations } from "drizzle-orm";
import { bigint, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";

export const deploymentApprovals = mysqlTable("deployment_approvals", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  deploymentId: varchar("deployment_id", { length: 128 }).notNull().unique(),
  approvedBy: varchar("approved_by", { length: 256 }).notNull(),
  approvedAt: bigint("approved_at", { mode: "number" }).notNull(),
  senderLogin: varchar("sender_login", { length: 256 }).notNull(),
});

export const deploymentApprovalsRelations = relations(deploymentApprovals, ({ one }) => ({
  deployment: one(deployments, {
    fields: [deploymentApprovals.deploymentId],
    references: [deployments.id],
  }),
}));
