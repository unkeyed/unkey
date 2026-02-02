import { relations } from "drizzle-orm";
import {
  bigint,
  boolean,
  index,
  int,
  json,
  mysqlTable,
  text,
  varchar,
} from "drizzle-orm/mysql-core";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const stripeConnectedAccounts = mysqlTable(
  "stripe_connected_accounts",
  {
    id: varchar("id", { length: 255 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    stripeAccountId: varchar("stripe_account_id", { length: 255 }).notNull(),
    accessTokenEncrypted: text("access_token_encrypted").notNull(),
    refreshTokenEncrypted: text("refresh_token_encrypted").notNull(),
    scope: varchar("scope", { length: 255 }).notNull(),
    connectedAt: bigint("connected_at", { mode: "number" }).notNull(),
    disconnectedAt: bigint("disconnected_at", { mode: "number" }),
    ...lifecycleDatesMigration,
  },
  (table) => [
    index("workspace_idx").on(table.workspaceId),
    index("stripe_account_idx").on(table.stripeAccountId),
  ],
);

export const pricingModels = mysqlTable(
  "pricing_models",
  {
    id: varchar("id", { length: 255 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    name: varchar("name", { length: 255 }).notNull(),
    currency: varchar("currency", { length: 3 }).notNull(),
    verificationUnitPrice: bigint("verification_unit_price", { mode: "number" }).notNull(),
    keyAccessUnitPrice: bigint("key_access_unit_price", { mode: "number" }).notNull().default(0),
    creditUnitPrice: bigint("credit_unit_price", { mode: "number" }).notNull().default(0),
    tieredPricing: json("tiered_pricing").$type<{
      tiers: Array<{
        upTo: number;
        unitPrice: number;
      }>;
    }>(),
    version: int("version").notNull().default(1),
    active: boolean("active").notNull().default(true),
    ...lifecycleDatesMigration,
  },
  (table) => [
    index("workspace_idx").on(table.workspaceId),
    index("workspace_currency_idx").on(table.workspaceId, table.currency),
  ],
);

export const billingEndUsers = mysqlTable(
  "billing_end_users",
  {
    id: varchar("id", { length: 255 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    externalId: varchar("external_id", { length: 255 }).notNull(),
    pricingModelId: varchar("pricing_model_id", { length: 255 }).notNull(),
    stripeCustomerId: varchar("stripe_customer_id", { length: 255 }).notNull(),
    stripeSubscriptionId: varchar("stripe_subscription_id", { length: 255 }),
    email: varchar("email", { length: 255 }),
    name: varchar("name", { length: 255 }),
    metadata: json("metadata").$type<Record<string, string>>(),
    ...lifecycleDatesMigration,
  },
  (table) => [
    index("workspace_external_idx").on(table.workspaceId, table.externalId),
    index("pricing_model_idx").on(table.pricingModelId),
    index("stripe_customer_idx").on(table.stripeCustomerId),
  ],
);

export const billingInvoices = mysqlTable(
  "billing_invoices",
  {
    id: varchar("id", { length: 255 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    endUserId: varchar("end_user_id", { length: 255 }).notNull(),
    stripeInvoiceId: varchar("stripe_invoice_id", { length: 255 }).notNull(),
    billingPeriodStart: bigint("billing_period_start", { mode: "number" }).notNull(),
    billingPeriodEnd: bigint("billing_period_end", { mode: "number" }).notNull(),
    verificationCount: bigint("verification_count", { mode: "number" }).notNull(),
    ratelimitCount: bigint("ratelimit_count", { mode: "number" }).notNull(),
    keyAccessCount: bigint("key_access_count", { mode: "number" }).notNull().default(0),
    creditsUsed: bigint("credits_used", { mode: "number" }).notNull().default(0),
    totalAmount: bigint("total_amount", { mode: "number" }).notNull(),
    currency: varchar("currency", { length: 3 }).notNull(),
    status: varchar("status", { length: 50 }).notNull(),
    ...lifecycleDatesMigration,
  },
  (table) => [
    index("workspace_idx").on(table.workspaceId),
    index("end_user_idx").on(table.endUserId),
    index("stripe_invoice_idx").on(table.stripeInvoiceId),
    index("billing_period_idx").on(table.billingPeriodStart, table.billingPeriodEnd),
  ],
);

export const billingTransactions = mysqlTable(
  "billing_transactions",
  {
    id: varchar("id", { length: 255 }).primaryKey(),
    invoiceId: varchar("invoice_id", { length: 255 }).notNull(),
    stripePaymentIntentId: varchar("stripe_payment_intent_id", { length: 255 }),
    amount: bigint("amount", { mode: "number" }).notNull(),
    currency: varchar("currency", { length: 3 }).notNull(),
    status: varchar("status", { length: 50 }).notNull(),
    failureReason: text("failure_reason"),
    createdAtM: bigint("created_at_m", { mode: "number" })
      .notNull()
      .default(0)
      .$defaultFn(() => Date.now()),
  },
  (table) => [
    index("invoice_idx").on(table.invoiceId),
    index("payment_intent_idx").on(table.stripePaymentIntentId),
  ],
);

export const stripeConnectedAccountsRelations = relations(stripeConnectedAccounts, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [stripeConnectedAccounts.workspaceId],
    references: [workspaces.id],
  }),
}));

export const pricingModelsRelations = relations(pricingModels, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [pricingModels.workspaceId],
    references: [workspaces.id],
  }),
  endUsers: many(billingEndUsers),
}));

export const billingEndUsersRelations = relations(billingEndUsers, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [billingEndUsers.workspaceId],
    references: [workspaces.id],
  }),
  pricingModel: one(pricingModels, {
    fields: [billingEndUsers.pricingModelId],
    references: [pricingModels.id],
  }),
  invoices: many(billingInvoices),
}));

export const billingInvoicesRelations = relations(billingInvoices, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [billingInvoices.workspaceId],
    references: [workspaces.id],
  }),
  endUser: one(billingEndUsers, {
    fields: [billingInvoices.endUserId],
    references: [billingEndUsers.id],
  }),
  transactions: many(billingTransactions),
}));

export const billingTransactionsRelations = relations(billingTransactions, ({ one }) => ({
  invoice: one(billingInvoices, {
    fields: [billingTransactions.invoiceId],
    references: [billingInvoices.id],
  }),
}));
