import { relations } from "drizzle-orm";
import { bigint, boolean, index, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * workspaceBilling holds the billing state for a workspace: its Stripe linkage,
 * the legacy API tier, the Compute (Deploy) plan entitlement, and the Compute
 * spend budget / spend-cap state.
 *
 * One row per workspace (keyed by workspace_id), mirroring the quota table. It
 * exists so billing concerns live in one place instead of accreting as columns
 * on the hot workspaces row. Stripe stays the source of truth for subscription
 * state; the columns here are local mirrors read by the dashboard, the deploy
 * gate, and the billing/spend-cap crons.
 */
export const workspaceBilling = mysqlTable(
  "workspace_billing",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    /**
     * workspaceId is the primary identifier for the billing record,
     * matching the ID of the workspace it belongs to.
     */
    workspaceId: varchar("workspace_id", { length: 256 }).notNull().unique(),

    /**
     * tier is the legacy API-product tier (Free/Pro/…), synced from Stripe by
     * the customer.subscription.* webhook. Distinct from plan, which is the
     * Compute (Deploy) plan.
     */
    tier: varchar("tier", { length: 256 }).default("Free"),

    // stripe
    stripeCustomerId: varchar("stripe_customer_id", { length: 256 }),

    /**
     * The API product's Stripe subscription. Every current prod customer's id
     * here is an API subscription, so the column keeps its name and existing
     * rows need no migration. Distinct from stripeDeploySubscriptionId, which
     * holds the Deploy product's subscription. Both subscriptions live on the
     * shared stripeCustomerId.
     */
    stripeSubscriptionId: varchar("stripe_subscription_id", { length: 256 }),

    /**
     * The Deploy (Compute) product's Stripe subscription, split out from the
     * API subscription so each product cancels as a native whole-subscription
     * operation. NULL means no Deploy subscription. Read by the ctrl deploy
     * billing machinery (invoice close, deprovision). Shares the same
     * stripeCustomerId as stripeSubscriptionId.
     */
    stripeDeploySubscriptionId: varchar("stripe_deploy_subscription_id", { length: 256 }),

    /**
     * Local mirror of the workspace's Unkey Deploy plan, synced from Stripe by
     * the customer.subscription.* webhook. NULL means no Deploy plan (cannot use
     * Deploy). Lets the deploy gate and dashboard read entitlement without
     * calling Stripe in the hot path. Stripe stays source of truth; this is a
     * cache. Distinct from tier, which is the legacy API-product tier.
     */
    plan: varchar("plan", { length: 64 }),

    /**
     * Manual Deploy entitlement override for internal / comped workspaces, owned
     * by us and never touched by the Stripe webhook. NULL = no override. When set
     * (to a plan value), the deploy gate treats the workspace as entitled even
     * without a paid plan. Kept separate from plan so that stays a pure Stripe
     * mirror.
     */
    planOverride: varchar("plan_override", { length: 64 }),

    /**
     * Monthly Compute (Deploy) spend budget in USD cents, set by workspace
     * admins in the dashboard. NULL = no budget. Email alerts fire at fixed
     * percentages of the budget (50/75/100); spendBudgetStop additionally stops
     * workloads when month-to-date usage spend reaches it.
     */
    spendBudgetCents: bigint("spend_budget_cents", {
      mode: "number",
      unsigned: true,
    }),
    spendBudgetStop: boolean("spend_budget_stop").notNull().default(false),

    /**
     * Written by the spend-cap check when it suspends or resumes a workspace's
     * compute. The dashboard reads it to show a "suspended by spend cap" state.
     * Lets the orchestrator find suspended workspaces even after their budget is
     * removed, so they still resume.
     */
    spendSuspended: boolean("spend_suspended").notNull().default(false),
    ...lifecycleDatesMigration,
  },
  (table) => ({
    // Back the spend-cap fleet scan (ListWorkspacesWithDeployBudget), which
    // filters spend_budget_cents IS NOT NULL OR spend_suspended = TRUE. MySQL has
    // no partial index; two single-column indexes let index-merge cover the OR.
    spendBudgetCentsIdx: index("spend_budget_cents_idx").on(table.spendBudgetCents),
    spendSuspendedIdx: index("spend_suspended_idx").on(table.spendSuspended),
    // Back the webhook subscription-id lookups, which are full scans today. One
    // index per subscription column, since the API and Deploy webhooks match on
    // their own column.
    stripeSubscriptionIdIdx: index("stripe_subscription_id_idx").on(table.stripeSubscriptionId),
    stripeDeploySubscriptionIdIdx: index("stripe_deploy_subscription_id_idx").on(
      table.stripeDeploySubscriptionId,
    ),
  }),
);

export const workspaceBillingRelations = relations(workspaceBilling, ({ one }) => ({
  workspace: one(workspaces, {
    relationName: "workspace_billing_relation",
    fields: [workspaceBilling.workspaceId],
    references: [workspaces.id],
  }),
}));
