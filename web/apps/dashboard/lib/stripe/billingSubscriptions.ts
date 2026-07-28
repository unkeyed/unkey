import { type Database, type Transaction, and, eq, schema } from "@unkey/db";

/**
 * A Stripe subscription belongs to exactly one Unkey product. After the billing
 * split each product owns a whole subscription, tracked as one row per
 * (workspace, product) in billing_subscriptions.
 */
export type SubscriptionProduct = "api" | "compute";

/**
 * Records (or repoints) the workspace's subscription for a product. Keyed on the
 * unique(workspace_id, product), so a resubscribe after a cancel overwrites the
 * id in place rather than leaving a stale row. Run inside the caller's
 * transaction so the id write commits atomically with the entitlement write it
 * accompanies (tier/plan on workspace_billing), matching the old single-row
 * update it replaces.
 */
export async function upsertBillingSubscription(
  db: Transaction | Database,
  params: { workspaceId: string; product: SubscriptionProduct; stripeSubscriptionId: string },
): Promise<void> {
  await db
    .insert(schema.billingSubscriptions)
    .values({
      workspaceId: params.workspaceId,
      product: params.product,
      stripeSubscriptionId: params.stripeSubscriptionId,
    })
    .onDuplicateKeyUpdate({ set: { stripeSubscriptionId: params.stripeSubscriptionId } });
}

/**
 * Removes the workspace's subscription row for a product. Replaces nulling the
 * old per-product column: a cancelled/deleted subscription leaves no row, so the
 * webhook lookup by stripe_subscription_id no longer resolves it.
 */
export async function deleteBillingSubscription(
  db: Transaction | Database,
  params: { workspaceId: string; product: SubscriptionProduct },
): Promise<void> {
  await db
    .delete(schema.billingSubscriptions)
    .where(
      and(
        eq(schema.billingSubscriptions.workspaceId, params.workspaceId),
        eq(schema.billingSubscriptions.product, params.product),
      ),
    );
}

/**
 * Flattens a workspace's subscription rows into the two id fields the app reads
 * everywhere (ctx.workspace.stripeSubscriptionId / stripeDeploySubscriptionId).
 * Keeps every read site working against the new table without threading the
 * product enum through them.
 */
export function subscriptionIdsByProduct(
  subscriptions: ReadonlyArray<{ product: SubscriptionProduct; stripeSubscriptionId: string }>,
): { stripeSubscriptionId: string | null; stripeDeploySubscriptionId: string | null } {
  let stripeSubscriptionId: string | null = null;
  let stripeDeploySubscriptionId: string | null = null;
  for (const s of subscriptions) {
    if (s.product === "api") {
      stripeSubscriptionId = s.stripeSubscriptionId;
    } else if (s.product === "compute") {
      stripeDeploySubscriptionId = s.stripeSubscriptionId;
    }
  }
  return { stripeSubscriptionId, stripeDeploySubscriptionId };
}
