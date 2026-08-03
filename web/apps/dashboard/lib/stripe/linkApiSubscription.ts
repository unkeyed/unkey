import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import Stripe from "stripe";
import { subscriptionIdsByProduct, upsertBillingSubscription } from "./billingSubscriptions";
import { deployBillingConfig, findApiItem } from "./deployBilling";
import { parseDeployPlan } from "./deployPlan";
import { validateAndParseQuotas } from "./productUtils";
import { setWorkspaceLimits } from "./setWorkspaceLimits";
import { isDeadSubscription } from "./subscriptionUtils";

export type LinkApiAudit = {
  actor: { type: "user" | "system"; id: string };
  location: string;
  userAgent: string | undefined;
};

export type LinkApiResult =
  | { ok: true; productName: string; alreadyLinked: boolean }
  | {
      ok: false;
      reason:
        | "session_not_found"
        | "workspace_not_found"
        | "forbidden"
        | "not_paid"
        | "not_active"
        | "no_api_plan"
        | "invalid_api_product"
        | "subscription_conflict";
      message: string;
    };

/** Links a paid API subscription Checkout and grants its configured limits. */
export async function linkApiSubscription(
  stripe: Stripe,
  input: { sessionId: string; expectedWorkspaceId: string; audit: LinkApiAudit },
): Promise<LinkApiResult> {
  let session: Stripe.Checkout.Session;
  try {
    session = await stripe.checkout.sessions.retrieve(input.sessionId);
  } catch (error) {
    if (error instanceof Stripe.errors.StripeError && error.code === "resource_missing") {
      return { ok: false, reason: "session_not_found", message: "Checkout session not found." };
    }
    throw error;
  }

  if (session.client_reference_id !== input.expectedWorkspaceId) {
    return {
      ok: false,
      reason: "forbidden",
      message: "Checkout session does not belong to this workspace.",
    };
  }
  if (session.metadata?.unkey_product !== "api") {
    return { ok: false, reason: "no_api_plan", message: "Checkout session is not for API." };
  }
  if (session.status !== "complete" || session.payment_status !== "paid") {
    return { ok: false, reason: "not_paid", message: "Checkout session is not paid." };
  }

  const stripeCustomerId =
    typeof session.customer === "string" ? session.customer : (session.customer?.id ?? null);
  const subscriptionId =
    typeof session.subscription === "string"
      ? session.subscription
      : (session.subscription?.id ?? null);
  if (!stripeCustomerId || !subscriptionId) {
    return {
      ok: false,
      reason: "not_paid",
      message: "Checkout session has no subscription or customer.",
    };
  }

  const sub = await stripe.subscriptions.retrieve(subscriptionId);
  const cancelling = Boolean(sub.cancel_at_period_end) || Boolean(sub.cancel_at);
  if ((sub.status !== "active" && sub.status !== "trialing") || cancelling) {
    return { ok: false, reason: "not_active", message: "Subscription is not active." };
  }

  const config = await deployBillingConfig();
  const apiItem = findApiItem(config, sub.items.data);
  const productRef = apiItem?.price.product;
  const productId =
    typeof productRef === "string"
      ? productRef
      : productRef && !productRef.deleted
        ? productRef.id
        : null;
  if (!productId) {
    return { ok: false, reason: "no_api_plan", message: "Subscription has no API plan." };
  }

  const e = stripeEnv();
  const allowedProductIds = e
    ? new Set([...e.STRIPE_PRODUCT_IDS_PRO, ...e.STRIPE_PRODUCT_IDS_ENTERPRISE])
    : null;
  if (!allowedProductIds?.has(productId)) {
    return {
      ok: false,
      reason: "invalid_api_product",
      message: "Subscription has an unsupported API plan.",
    };
  }

  const product = await stripe.products.retrieve(productId);
  const quotas = validateAndParseQuotas(product);
  if (!quotas.valid) {
    return {
      ok: false,
      reason: "invalid_api_product",
      message: "API plan is missing required quota configuration.",
    };
  }

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq: eqFn, isNull }) =>
      and(eqFn(table.id, input.expectedWorkspaceId), isNull(table.deletedAtM)),
    columns: { id: true },
    with: {
      billing: {
        columns: { tier: true },
      },
      billingSubscriptions: {
        columns: { product: true, stripeSubscriptionId: true },
      },
    },
  });
  if (!ws) {
    return { ok: false, reason: "workspace_not_found", message: "Workspace not found." };
  }

  const recordedSubscriptionId = subscriptionIdsByProduct(
    ws.billingSubscriptions ?? [],
  ).stripeSubscriptionId;
  if (recordedSubscriptionId === subscriptionId) {
    if (ws.billing?.tier === product.name) {
      return { ok: true, productName: product.name, alreadyLinked: true };
    }
  } else if (recordedSubscriptionId) {
    const recorded = await stripe.subscriptions
      .retrieve(recordedSubscriptionId)
      .catch((error: unknown) => {
        if (error instanceof Stripe.errors.StripeError && error.code === "resource_missing") {
          return null;
        }
        throw error;
      });
    if (recorded && !isDeadSubscription(recorded)) {
      return {
        ok: false,
        reason: "subscription_conflict",
        message: "Workspace already has a different API subscription.",
      };
    }
  }

  const { requestsPerMonth, logsRetentionDays, auditLogsRetentionDays } = quotas;
  await db.transaction(async (tx) => {
    await tx
      .update(schema.workspaceBilling)
      .set({ stripeCustomerId, tier: product.name })
      .where(eq(schema.workspaceBilling.workspaceId, ws.id));
    await upsertBillingSubscription(tx, {
      workspaceId: ws.id,
      product: "api",
      stripeSubscriptionId: subscriptionId,
    });
    await setWorkspaceLimits(tx, {
      workspaceId: ws.id,
      plan:
        parseDeployPlan(ws.billing?.planOverride ?? null) ??
        parseDeployPlan(ws.billing?.plan ?? null),
      preserveApiLimits: true,
      limitUpdate: {
        apiBillableOperationsCountMaxPerMonth: requestsPerMonth,
        logsRetentionDaysMax: logsRetentionDays,
        logsAuditRetentionDaysMax: auditLogsRetentionDays,
        teamEnabled: true,
      },
    });
    await insertAuditLogs(tx, {
      workspaceId: ws.id,
      actor: input.audit.actor,
      event: "workspace.update",
      description: `Subscribed to ${product.name} plan via checkout.`,
      resources: [],
      context: { location: input.audit.location, userAgent: input.audit.userAgent },
    });
  });

  // Checkout's selected card must replace the known-bad customer default so
  // future subscriptions and renewals use the card that just succeeded.
  const paymentMethod =
    typeof sub.default_payment_method === "string"
      ? sub.default_payment_method
      : sub.default_payment_method?.id;
  if (paymentMethod) {
    await stripe.customers
      .update(stripeCustomerId, { invoice_settings: { default_payment_method: paymentMethod } })
      .catch((error: unknown) => {
        console.error("Failed to update customer default after API checkout", {
          workspaceId: ws.id,
          subscriptionId,
          error: error instanceof Error ? error.message : error,
        });
      });
  }

  return { ok: true, productName: product.name, alreadyLinked: false };
}
