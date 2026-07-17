import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import Stripe from "stripe";
import { type DeployPlan, detectDeployPlan } from "./deployPlan";
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
  | "no_deploy_plan"
  | "subscription_conflict";

export type LinkDeployResult =
  | { ok: true; plan: DeployPlan; alreadyLinked: boolean }
  | { ok: false; reason: LinkDeployFailure; message: string };

/**
 * Links a subscription-mode Compute checkout onto its workspace: verifies the
 * session belongs to the workspace and was paid, that the resulting
 * subscription is live and carries a recognized Deploy plan, then writes
 * `stripeCustomerId` + `stripeSubscriptionId` + `deployPlan` optimistically
 * (mirroring subscribeDeploy; the customer.subscription.* webhook then derives
 * the same value and no-ops).
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
  let session: Stripe.Checkout.Session;
  try {
    session = await stripe.checkout.sessions.retrieve(input.sessionId);
  } catch (err) {
    // Only a genuinely missing session is a permanent "not found". A transient
    // Stripe failure (network, 429, 5xx) must propagate so the webhook returns
    // 500 and Stripe retries — swallowing it here would ack the event and leave
    // a paid subscription orphaned from a purely transient cause.
    if (err instanceof Stripe.errors.StripeError && err.code === "resource_missing") {
      return { ok: false, reason: "session_not_found", message: "Checkout session not found." };
    }
    throw err;
  }

  // Ownership: the session must have been created for this workspace. This is
  // the same guard updateWorkspaceStripeCustomer uses to stop an attacker
  // binding their session to a victim workspace.
  if (session.client_reference_id !== input.expectedWorkspaceId) {
    return {
      ok: false,
      reason: "forbidden",
      message: "Checkout session does not belong to this workspace.",
    };
  }

  // Payment gate: only a completed, paid session may grant entitlement.
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
  // Only a live subscription grants a plan. Mirrors subscribeDeploy's
  // active/trialing guard so an incomplete/past_due first charge cannot
  // entitle the workspace.
  if (sub.status !== "active" && sub.status !== "trialing") {
    return { ok: false, reason: "not_active", message: "Subscription is not active." };
  }

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
  });
  if (!ws) {
    return { ok: false, reason: "workspace_not_found", message: "Workspace not found." };
  }

  // Idempotency + conflict: re-entry (webhook + /success, refresh, redelivery)
  // for the same subscription is a success no-op; a *different* LIVE existing
  // subscription is a hard failure so we never orphan a live one by repointing.
  // A dead recorded subscription (cancelDeploy cancels a Compute-only
  // subscription outright, and the deleted-webhook that clears the column may
  // lag) is safe to repoint away from — refusing would strand this checkout's
  // paid subscription instead.
  if (ws.stripeSubscriptionId === subscriptionId) {
    if (ws.deployPlan === plan) {
      return { ok: true, plan, alreadyLinked: true };
    }
  } else if (ws.stripeSubscriptionId) {
    const recorded = await stripe.subscriptions
      .retrieve(ws.stripeSubscriptionId)
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
      .update(schema.workspaces)
      .set({ stripeCustomerId, stripeSubscriptionId: subscriptionId, deployPlan: plan })
      .where(eq(schema.workspaces.id, ws.id));
    await insertAuditLogs(tx, {
      workspaceId: ws.id,
      actor: input.audit.actor,
      event: "workspace.update",
      description: `Subscribed to Compute ${plan} plan via checkout.`,
      resources: [],
      context: { location: input.audit.location, userAgent: input.audit.userAgent },
    });
  });

  // Checkout attaches the card to the customer but records it as the
  // SUBSCRIPTION's default payment method, not the customer's. Later flows
  // that create a fresh subscription (cancel-then-resubscribe, the API plan)
  // only consult the customer default, so mirror it over — best-effort and
  // only when the customer has none, because the link above is already
  // committed and must not fail on this.
  const paymentMethod =
    typeof sub.default_payment_method === "string"
      ? sub.default_payment_method
      : sub.default_payment_method?.id;
  if (paymentMethod) {
    try {
      const customer = await stripe.customers.retrieve(stripeCustomerId);
      if (
        !customer.deleted &&
        !customer.invoice_settings?.default_payment_method &&
        !customer.default_source
      ) {
        await stripe.customers.update(stripeCustomerId, {
          invoice_settings: { default_payment_method: paymentMethod },
        });
      }
    } catch (err) {
      console.error("Failed to set customer default payment method after Compute link", {
        workspaceId: ws.id,
        error: err instanceof Error ? err.message : err,
      });
    }
  }

  return { ok: true, plan, alreadyLinked: false };
}
