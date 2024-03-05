import { relations } from "drizzle-orm";
import {
  boolean,
  datetime,
  float,
  index,
  json,
  mysqlEnum,
  mysqlTable,
  varchar,
} from "drizzle-orm/mysql-core";
import { workspaces } from "./workspaces";

export const budgets = mysqlTable(
  "budgets",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 })
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),
    name: varchar("name", { length: 256 }),
    type: mysqlEnum("type", ["soft", "hard"]).notNull(),
    /**
     * Sets if the budget is enabled or not.
     */
    enabled: boolean("enabled").default(true).notNull(),
    fixedAmount: float("fixed_amount").notNull(),
    data: json("data")
      .$type<{
        /**
         * Additional emails to notify.
         */
        additionalEmails?: string[];

        /**
         * Webhook URL to POST trigger when metered resources reached set amount.
         * @dev TODO: Not used currently.
         */
        webhookUrl?: string;
      }>()
      .notNull(),
    createdAt: datetime("created_at", { mode: "date", fsp: 3 }),
  },
  (table) => ({
    workspaceId: index("workspace_id_idx").on(table.workspaceId),
    // TODO: Should index by `enabled`?
  }),
);

export const budgetsRelations = relations(budgets, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [budgets.workspaceId],
    references: [workspaces.id],
  }),
}));
