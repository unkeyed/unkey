import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import Stripe from "stripe";
import { subscriptionIdsByProduct, upsertBillingSubscription } from "./billingSubscriptions";
import { deployBillingConfig, findApiItem } from "./deployBilling";
import { type DeployPlan, detectDeployPlan } from "./deployPlan";
import { validateAndParseQuotas } from "./productUtils";
import { isDeadSubscription } from "./subscriptionUtils";

/**
 * Audit provenance for the write. The tRPC caller passes the acting user; the
 * webhook passes the Stripe system actor with empty request context.
 */
export type LinkDeployAudit = {
  actor: { type: "user" | "system"; id: string };
  location: string;
  userAgent: string | undefined;
};

/**
 * Why a link attempt did not write. Callers map these: `forbidden` ->
 * FORBIDDEN, `session_not_found`/`workspace_not_found` -> NOT_FOUND, the rest
 * -> PRECONDITION_FAILED (tRPC) or a logged no-op (webhook).
 */
export type LinkDeployFailure =
  | "session_not_found"
  | "workspace_not_found"
  | "forbidden"
  | "not_paid"
  | "not_active"
  | "no_api_plan"
  | "invalid_api_product"
  | "no_deploy_plan"
  | "subscription_conflict";

export type LinkDeployResult =
  | { ok: true; plan: DeployPlan; alreadyLinked: boolean }
  | { ok: false; reason: LinkDeployFailure; message: string };

export type LinkApiResult =
  | { ok: true; productName: string; alreadyLinked: boolean }
  | { ok: false; reason: LinkDeployFailure; message: string };

type PaidCheckout = {
  stripeCustomerId: string;
  subscriptionId: string;
  subscription: Stripe.Subscription;
};

async function resolvePaidCheckout(
  stripe: Stripe,
  input: {
    sessionId: string;
    expectedWorkspaceId: string;
    expectedProduct?: "api";
  },
): Promise<{ ok: true; checkout: PaidCheckout } | { ok: false; reason: LinkDeployFailure; message: string }> {
  let session: Stripe.Checkout.Session;
  try {
    session = await stripe.checkout.sessions.retrieve(input.sessionId);
  } catch (err) {
    if (err instanceof Stripe.errors.StripeError && err.code === "resource_missing") {
      return { ok: false, reason: "session_not_found", message: "Checkout session not found." };
    }
    throw err;
  }

  if (session.client_reference_id !== input.expectedWorkspaceId) {
    return {
      ok: false,
      reason: "forbidden",
      message: "Checkout session does not belong to this workspace.",
    };
  }
  if (input.expectedProduct && session.metadata?.unkey_product !== input.expectedProduct) {
    return {
      ok: false,
      reason: "no_api_plan",
      message: "Checkout session is not for an API plan.",
    };
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

  const subscription = await stripe.subscriptions.retrieve(subscriptionId);
  const cancelling =
    Boolean(subscription.cancel_at_period_end) || Boolean(subscription.cancel_at);
  if (
    (subscription.status !== "active" && subscription.status !== "trialing") ||
    cancelling
  ) {
    return { ok: false, reason: "not_active", message: "Subscription is not active." };
  }

  return {
    ok: true,
    checkout: { stripeCustomerId, subscriptionId, subscription },
  };
}

/** Checkout stores its selected card on the subscription; mirror it to the
 * customer so later subscriptions and plan changes can reuse it. Linking has
 * already committed when this runs, so a Stripe failure is best-effort only. */
async function mirrorCheckoutPaymentMethod(
  stripe: Stripe,
  input: {
    workspaceId: string;
    stripeCustomerId: string;
    sub: Stripe.Subscription;
    product: "API" | "Compute";
  },
): Promise<void> {
  const paymentMethod =
    typeof input.sub.default_payment_method === "string"
      ? input.sub.default_payment_method
      : input.sub.default_payment_method?.id;
  if (!paymentMethod) {
    return;
  }

  try {
    const customer = await stripe.customers.retrieve(input.stripeCustomerId);
    if (
      !customer.deleted &&
      !customer.invoice_settings?.default_payment_method &&
      !customer.default_source
    ) {
      await stripe.customers.update(input.stripeCustomerId, {
        invoice_settings: { default_payment_method: paymentMethod },
      });
    }
  } catch (err) {
    console.error(`Failed to set customer default payment method after ${input.product} link`, {
      workspaceId: input.workspaceId,
      error: err instanceof Error ? err.message : err,
    });
  }
}

/**
 * Links a paid subscription-mode API checkout onto its workspace. Checkout
 * owns first-payment recovery (CVC recollection and 3DS); this function owns
 * the entitlement boundary and only writes the paid tier after Stripe reports
 * both the session and subscription as complete and active.
 *
 * Shared by /success and checkout.session.completed, so it is deliberately
 * idempotent. A different live API subscription is never overwritten; a dead
 * recorded subscription may be replaced on a later subscribe cycle.
 */
export async function linkApiSubscription(
  stripe: Stripe,
  input: { sessionId: string; expectedWorkspaceId: string; audit: LinkDeployAudit },
): Promise<LinkApiResult> {
  const resolved = await resolvePaidCheckout(stripe, { ...input, expectedProduct: "api" });
  if (!resolved.ok) {
    return resolved;
  }
  const { stripeCustomerId, subscriptionId, subscription: sub } = resolved.checkout;

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
  if (
    !quotas.valid ||
    quotas.requestsPerMonth === undefined ||
    quotas.logsRetentionDays === undefined ||
    quotas.auditLogsRetentionDays === undefined
  ) {
    return {
      ok: false,
      reason: "invalid_api_product",
      message: "API plan is missing required quota configuration.",
    };
  }

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq: eqFn, isNull }) =>
      and(eqFn(table.id, input.expectedWorkspaceId), isNull(table.deletedAtM)),
    with: { billing: true, billingSubscriptions: true },
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
      .catch((err: unknown) => {
        if (err instanceof Stripe.errors.StripeError && err.code === "resource_missing") {
          return null;
        }
        throw err;
      });
    if (recorded && !isDeadSubscription(recorded)) {
      return {
        ok: false,
        reason: "subscription_conflict",
        message: "Workspace already has a different API subscription.",
      };
    }
  }

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
    await tx
      .insert(schema.quotas)
      .values({
        workspaceId: ws.id,
        requestsPerMonth: quotas.requestsPerMonth,
        logsRetentionDays: quotas.logsRetentionDays,
        auditLogsRetentionDays: quotas.auditLogsRetentionDays,
        team: true,
      })
      .onDuplicateKeyUpdate({
        set: {
          requestsPerMonth: quotas.requestsPerMonth,
          logsRetentionDays: quotas.logsRetentionDays,
          auditLogsRetentionDays: quotas.auditLogsRetentionDays,
          team: true,
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

  await mirrorCheckoutPaymentMethod(stripe, {
    workspaceId: ws.id,
    stripeCustomerId,
    sub,
    product: "API",
  });

  return { ok: true, productName: product.name, alreadyLinked: false };
}

/**
 * Links a subscription-mode Compute checkout onto its workspace: verifies the
 * session belongs to the workspace and was paid, that the resulting
 * subscription is live and carries a recognized Deploy plan, then writes
 * `stripeCustomerId` + `stripeDeploySubscriptionId` + `plan` to workspace_billing
 * optimistically; the customer.subscription.* webhook then derives the same
 * value and no-ops.
 *
 * Shared by `/success` (fast-path when the user returns) and the
 * `checkout.session.completed` webhook (guaranteed, fires even if the user
 * never returns). Both entry points call this with the same session, so it is
 * idempotent: a matching already-linked subscription is a success no-op, and a
 * *different* existing subscription is a hard failure (never repoint/orphan a
 * live subscription).
 *
 * The session and subscription are resolved server-side; no id is trusted from
 * the caller beyond the session id and the workspace id to check ownership
 * against. Payment/status are verified here because detectDeployPlan keys only
 * on price metadata, never on whether the subscription was actually paid.
 */
export async function linkDeploySubscription(
  stripe: Stripe,
  input: { sessionId: string; expectedWorkspaceId: string; audit: LinkDeployAudit },
): Promise<LinkDeployResult> {
  const resolved = await resolvePaidCheckout(stripe, input);
  if (!resolved.ok) {
    return resolved;
  }
  const { stripeCustomerId, subscriptionId, subscription: sub } = resolved.checkout;

  const plan = detectDeployPlan(sub);
  if (!plan) {
    return {
      ok: false,
      reason: "no_deploy_plan",
      message: "Subscription does not carry a Compute plan.",
    };
  }

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq: eqFn, isNull }) =>
      and(eqFn(table.id, input.expectedWorkspaceId), isNull(table.deletedAtM)),
    with: { billing: true, billingSubscriptions: true },
  });
  if (!ws) {
    return { ok: false, reason: "workspace_not_found", message: "Workspace not found." };
  }

  const recordedSubscriptionId = subscriptionIdsByProduct(
    ws.billingSubscriptions ?? [],
  ).stripeDeploySubscriptionId;

  // Idempotency + conflict: re-entry (webhook + /success, refresh, redelivery)
  // for the same subscription is a success no-op; a *different* LIVE existing
  // subscription is a hard failure so we never orphan a live one by repointing.
  // A dead recorded subscription (cancelDeploy cancels the Compute subscription
  // outright, and the deleted-webhook that clears the column may lag) is safe to
  // repoint away from — refusing would strand this checkout's paid subscription.
  if (recordedSubscriptionId === subscriptionId) {
    if (ws.billing?.plan === plan) {
      return { ok: true, plan, alreadyLinked: true };
    }
  } else if (recordedSubscriptionId) {
    const recorded = await stripe.subscriptions
      .retrieve(recordedSubscriptionId)
      .catch((err: unknown) => {
        if (err instanceof Stripe.errors.StripeError && err.code === "resource_missing") {
          return null;
        }
        throw err;
      });
    if (recorded && !isDeadSubscription(recorded)) {
      return {
        ok: false,
        reason: "subscription_conflict",
        message: "Workspace already has a different subscription.",
      };
    }
  }

  await db.transaction(async (tx) => {
    await tx
      .update(schema.workspaceBilling)
      .set({ stripeCustomerId, plan })
      .where(eq(schema.workspaceBilling.workspaceId, ws.id));
    await upsertBillingSubscription(tx, {
      workspaceId: ws.id,
      product: "compute",
      stripeSubscriptionId: subscriptionId,
    });
    await insertAuditLogs(tx, {
      workspaceId: ws.id,
      actor: input.audit.actor,
      event: "workspace.update",
      description: `Subscribed to Compute ${plan} plan via checkout.`,
      resources: [],
      context: { location: input.audit.location, userAgent: input.audit.userAgent },
    });
  });

  await mirrorCheckoutPaymentMethod(stripe, {
    workspaceId: ws.id,
    stripeCustomerId,
    sub,
    product: "Compute",
  });

  return { ok: true, plan, alreadyLinked: false };
}
