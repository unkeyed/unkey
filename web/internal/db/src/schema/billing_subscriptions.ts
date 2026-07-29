import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { workspaces } from "./workspaces";

/**
 * billingSubscriptions holds one row per (workspace, product) Stripe
 * subscription. It replaces the stripe_subscription_id / stripe_deploy_
 * subscription_id columns that used to sit on workspace_billing: after the
 * billing split each product owns a whole subscription, so modelling them as
 * rows keyed by product means the webhook resolves a subscription to its
 * product with a single unique-index lookup, and a future product is a new
 * enum value rather than another column, index, and migration.
 *
 * The customer id, plan/tier entitlement, and spend state stay on
 * workspace_billing: those are per-workspace, not per-subscription. Stripe
 * remains the source of truth for subscription state; the id here is the local
 * handle used to route webhooks and drive the billing crons.
 */
export const billingSubscriptions = mysqlTable(
  "billing_subscriptions",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    /**
     * The workspace this subscription belongs to. Both products share the
     * workspace's single Stripe customer (workspace_billing.stripe_customer_id).
     */
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 256 }).notNull(),

    /**
     * Which Unkey product the subscription bills. "api" is the legacy API tier
     * (Free/Pro/…), "compute" is the Compute plan. The workspace's entitlement
     * mirrors (tier, plan) still live on workspace_billing.
     */
    product: mysqlEnum("product", ["api", "compute"]).notNull(),

    /**
     * The Stripe subscription id. Unique across the table so the webhook can
     * resolve any incoming subscription event straight to its (workspace,
     * product) without inspecting the subscription's items.
     */
    stripeSubscriptionId: caseSensitiveVarchar("stripe_subscription_id", { length: 256 })
      .notNull()
      .unique(),

    // No deleted_at_m: a cancel/delete hard-deletes the row (see
    // deleteBillingSubscription), so there is no soft-delete state to keep.
    // created_at records when the subscription was first linked; updated_at
    // bumps when it is repointed on a resubscribe.
    createdAt: bigint("created_at", { mode: "number" })
      .notNull()
      .default(0)
      .$defaultFn(() => Date.now()),
    updatedAt: bigint("updated_at", { mode: "number" }).$onUpdateFn(() => Date.now()),
  },
  (table) => [
    index("workspace_id_idx").on(table.workspaceId),
    // At most one live subscription per product per workspace, mirroring the
    // single-column-per-product model it replaces.
    uniqueIndex("unique_product_per_workspace").on(table.workspaceId, table.product),
  ],
);

export const billingSubscriptionsRelations = relations(billingSubscriptions, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [billingSubscriptions.workspaceId],
    references: [workspaces.id],
  }),
}));
