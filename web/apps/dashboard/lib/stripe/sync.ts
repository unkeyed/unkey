import { db, eq, schema } from "@/lib/db";
import { freeTierQuotas } from "@/lib/quotas";
import type Stripe from "stripe";

export async function syncSubscriptionFromStripe(
  stripe: Stripe,
  workspaceId: string,
): Promise<void> {
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.id, workspaceId), isNull(table.deletedAtM)),
  });

  if (!workspace?.stripeSubscriptionId) {
    return;
  }

  const subscription = await stripe.subscriptions.retrieve(workspace.stripeSubscriptionId);

  const priceId = subscription.items.data[0]?.price?.id;
  if (!priceId) {
    return;
  }

  const price = await stripe.prices.retrieve(priceId);
  if (!price.product) {
    return;
  }

  const product = await stripe.products.retrieve(
    typeof price.product === "string" ? price.product : price.product.id,
  );

  const requestsPerMonth = Number.parseInt(product.metadata.quota_requests_per_month || "0");
  if (!product.metadata.quota_requests_per_month) {
    console.warn(`Product ${product.id} missing quota_requests_per_month metadata`);
  }
  const logsRetentionDays = Number.parseInt(product.metadata.quota_logs_retention_days || "0");
  const auditLogsRetentionDays = Number.parseInt(
    product.metadata.quota_audit_logs_retention_days || "0",
  );

  const isFailedPayment = subscription.status === "past_due" || subscription.status === "unpaid";
  const wasFailedPayment = workspace.paymentFailedAt !== null;

  // Map Stripe status to allowed database values
  type SubscriptionStatus = "active" | "past_due" | "canceled" | "unpaid" | "trialing" | "incomplete" | "incomplete_expired";
  const allowedStatuses: readonly string[] = ["active", "past_due", "canceled", "unpaid", "trialing", "incomplete", "incomplete_expired"];
  
  function isSubscriptionStatus(value: unknown): value is SubscriptionStatus {
    return typeof value === "string" && allowedStatuses.includes(value);
  }
  
  const subscriptionStatus: SubscriptionStatus = isSubscriptionStatus(subscription.status)
    ? subscription.status
    : "canceled";

  await db.transaction(async (tx) => {
    await tx
      .update(schema.workspaces)
      .set({
        tier: product.name,
        subscriptionStatus,
        paymentFailedAt: isFailedPayment && !wasFailedPayment ? Date.now() : wasFailedPayment && !isFailedPayment ? null : workspace.paymentFailedAt,
      })
      .where(eq(schema.workspaces.id, workspaceId));

    await tx
      .insert(schema.quotas)
      .values({
        workspaceId,
        requestsPerMonth,
        logsRetentionDays,
        auditLogsRetentionDays,
        team: true,
      })
      .onDuplicateKeyUpdate({
        set: {
          requestsPerMonth,
          logsRetentionDays,
          auditLogsRetentionDays,
          team: true,
        },
      });
  });
}

export async function syncCanceledSubscription(workspaceId: string): Promise<void> {
  await db.transaction(async (tx) => {
    await tx
      .update(schema.workspaces)
      .set({
        stripeSubscriptionId: null,
        tier: "Free",
        subscriptionStatus: "canceled",
        paymentFailedAt: null,
        paymentFailureNotifiedAt: null,
      })
      .where(eq(schema.workspaces.id, workspaceId));

    await tx
      .insert(schema.quotas)
      .values({
        workspaceId,
        ...freeTierQuotas,
      })
      .onDuplicateKeyUpdate({
        set: freeTierQuotas,
      });
  });
}
