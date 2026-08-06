import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { insertAuditLogs } from "@/lib/audit";
import { deactivateNonCreatorMemberships } from "@/lib/auth/deactivateNonCreatorMemberships";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { formatPrice } from "@/lib/fmt";
import { deleteBillingSubscription } from "@/lib/stripe/billingSubscriptions";
import {
  type ComputeLifecycleAlert,
  computeCreatedAlert,
  computeUpdatedAlert,
} from "@/lib/stripe/computeAlerts";
import { deployBillingConfig, findApiItem } from "@/lib/stripe/deployBilling";
import { grantDeployCreditsForInvoice } from "@/lib/stripe/deployCredits";
import { deployPlanGrantsTeam, detectDeployPlan, parseDeployPlan } from "@/lib/stripe/deployPlan";
import { linkApiSubscription } from "@/lib/stripe/linkApiSubscription";
import { linkDeploySubscription } from "@/lib/stripe/linkDeploySubscription";
import { isPaymentRecovery, isPaymentRecoveryUpdate } from "@/lib/stripe/paymentUtils";
import { validateAndParseQuotas } from "@/lib/stripe/productUtils";
import { setWorkspaceLimits } from "@/lib/stripe/setWorkspaceLimits";
import {
  isAutomatedBillingRenewal,
  isCardUpdateOnly,
  isPaymentFailureRelatedUpdate,
} from "@/lib/stripe/subscriptionUtils";
import { keepsTeamAfterDelete } from "@/lib/stripe/webhookRouting";
import {
  alertCustomerLifecycle,
  alertInvalidProductQuotaMetadata,
  alertOrphanedDeploySubscription,
  alertPaymentFailed,
  alertPaymentRecovered,
} from "@/lib/utils/slackAlerts";
import Stripe from "stripe";

/**
 * Mirrors a subscription's Deploy plan onto its workspace row, writing only when
 * it changed so the common renewal case (plan unchanged) does no DB write.
 * Stripe stays the source of truth; workspace_billing.plan is the cache the
 * deploy gate and dashboard read without a Stripe call in the hot path.
 *
 * A subscription set to cancel is treated as plan-ending and mirrors null:
 * cancelDeploy tears down compute and clears the entitlement immediately (not at
 * the period boundary), yet Stripe keeps the plan-fee item on the still-active
 * subscription and bills to the boundary. Without this, the customer.subscription
 * .updated event that the cancel itself fires would re-detect that item and
 * resurrect the plan the cancel just cleared, leaving the workspace unable to
 * resubscribe (subscribeDeploy requires plan IS NULL). A resume clears cancel_at
 * and the next update re-mirrors the live plan. This mirrors the API side, whose
 * updated handler already returns early on a cancelling subscription.
 */
async function mirrorDeployPlan(
  billing: { workspaceId: string; plan: string | null; tier: string | null },
  orgId: string,
  sub: Stripe.Subscription,
): Promise<void> {
  const cancelling = Boolean(sub.cancel_at_period_end) || Boolean(sub.cancel_at);
  const plan = cancelling ? null : detectDeployPlan(sub);
  const changed = plan !== billing.plan;
  if (changed) {
    const preserveApiLimits = (billing.tier ?? "Free") !== "Free";
    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaceBilling)
        .set({ plan })
        .where(eq(schema.workspaceBilling.workspaceId, billing.workspaceId));
      await setWorkspaceLimits(tx, {
        workspaceId: billing.workspaceId,
        plan,
        preserveApiLimits,
      });
    });

    if (!preserveApiLimits && deployPlanGrantsTeam(billing.plan) && !deployPlanGrantsTeam(plan)) {
      await deactivateNonCreatorMemberships(orgId);
    }
  }
}

/**
 * Sends the operational Slack alert for a Compute (Deploy) subscription lifecycle
 * event, resolving the customer email/name the alert renders. Best-effort: a
 * failed customer lookup is logged and swallowed, never thrown, so it cannot fail
 * the webhook whose real work (the plan mirror) has already committed. The alert
 * post itself never throws (postToSlack logs its own failures). A null descriptor
 * (an event that warrants no alert) is a no-op.
 */
async function sendComputeAlert(
  stripe: Stripe,
  sub: Stripe.Subscription,
  alert: ComputeLifecycleAlert | null,
  ws: { id: string; name: string } | null,
): Promise<void> {
  if (!alert || !sub.customer) {
    return;
  }

  let customer: Stripe.Customer | Stripe.DeletedCustomer;
  try {
    customer = await stripe.customers.retrieve(
      typeof sub.customer === "string" ? sub.customer : sub.customer.id,
    );
  } catch (err) {
    console.error("Failed to retrieve customer for Compute subscription alert:", {
      subscriptionId: sub.id,
      error: err instanceof Error ? err.message : err,
    });
    return;
  }

  if (customer.deleted || !customer.email) {
    return;
  }

  // Common customer/workspace facts every Compute lifecycle alert renders. ws is null when a
  // created event races ahead of the checkout link that writes the billing_subscriptions row;
  // the workspace fields are simply omitted from the alert in that case.
  const base = {
    name: customer.name || "Unknown",
    email: customer.email,
    workspaceId: ws?.id,
    workspaceName: ws?.name,
    stripeCustomerId: customer.id,
    livemode: customer.livemode,
  };

  switch (alert.type) {
    case "created":
      await alertCustomerLifecycle({
        ...base,
        action: "signup",
        product: alert.product,
        price: alert.price,
      });
      break;
    case "cancelling":
      await alertCustomerLifecycle({
        ...base,
        action: "cancelling",
        product: alert.product,
        price: alert.price,
      });
      break;
    case "updated":
      await alertCustomerLifecycle({
        ...base,
        action: alert.changeType === "downgraded" ? "downgrade" : "upgrade",
        product: alert.product,
        previousProduct: alert.previousTier,
        price: alert.price,
      });
      break;
  }
}

/**
 * Whether a subscription Stripe told us about is still capable of billing.
 *
 * Used to decide whether an unresolvable workspace is a real orphan or just a
 * redelivered terminal event. `canceled` and `incomplete_expired` are Stripe's
 * end states: nothing further is charged, so losing the local link is untidy but
 * costs nothing. Any other status on a subscription we cannot resolve is money
 * moving with no workspace attached, which is worth waking someone for.
 */
function subscriptionStillBilling(sub: Stripe.Subscription): boolean {
  return sub.status !== "canceled" && sub.status !== "incomplete_expired";
}

/**
 * Tears down Compute when a subscription becomes cancelling, before
 * [[mirrorDeployPlan]] clears the plan.
 *
 * Ordering is load-bearing. ctrl's DeprovisionCompute keys its idempotency guard
 * on `workspace_billing.plan` still being set, and deliberately tears down
 * before clearing it. mirrorDeployPlan writes that same column to null on any
 * cancelling subscription, which is correct for the dashboard flow (cancelDeploy
 * has already deprovisioned, so the null is just the mirror catching up) but not
 * for a cancel made on the Stripe side, in the customer portal or the Stripe
 * dashboard. There our tRPC endpoint never runs, so the plan went to null with no
 * teardown, and by the time `customer.subscription.deleted` arrived its
 * `if (billing.plan)` gate saw null and skipped teardown for good. Workloads kept
 * running with no subscription, and nothing billed them, because every billable
 * query and the spend cap all require plan IS NOT NULL.
 *
 * Calling deprovision here restores the invariant those guards assume: by the
 * time plan is null, teardown has been dispatched. On the dashboard path this is
 * a no-op, since the plan is already null and ctrl returns early. Cancel means
 * immediate teardown either way, which is the semantics cancelDeploy already
 * chose (billing runs to the period boundary, no refund).
 *
 * Failures propagate so the caller returns 500 and Stripe redelivers: dropping
 * this is how compute ends up running unbilled.
 */
async function deprovisionOnCancel(
  billing: { workspaceId: string; plan: string | null },
  sub: Stripe.Subscription,
): Promise<void> {
  const cancelling = Boolean(sub.cancel_at_period_end) || Boolean(sub.cancel_at);
  if (!cancelling || !billing.plan) {
    return;
  }

  const ctrl = createCtrlClient(DeployService);
  await ctrl.deprovisionCompute({ workspaceId: billing.workspaceId });
}

/**
 * Links a subscription-mode API or Compute checkout to its workspace via the
 * product's shared linker. Called by `checkout.session.completed` and
 * `checkout.session.async_payment_succeeded` events (the latter fires when a
 * delayed-notification payment clears after `completed` reported it unpaid).
 *
 * Returns 200 for anything a retry cannot fix (not a subscription checkout,
 * missing workspace ref, or a linker rejection); lets transient Stripe/DB
 * errors from the linker propagate so the caller returns 500 and Stripe
 * retries. A `subscription_conflict` means a paid, live subscription exists
 * that will never link (a race minted a duplicate) — it bills until an operator
 * intervenes, so page a human rather than only logging.
 */
async function linkCheckoutSession(
  stripe: Stripe,
  session: Stripe.Checkout.Session,
  eventId: string,
): Promise<Response> {
  if (session.mode !== "subscription" || !session.subscription) {
    return new Response("OK", { status: 200 });
  }
  const productTag = session.metadata?.unkey_product;
  if (productTag !== "api" && productTag !== "compute") {
    return new Response("OK", { status: 200 });
  }
  const subscriptionId =
    typeof session.subscription === "string" ? session.subscription : session.subscription.id;
  const product = productTag === "api" ? "API" : "Compute";
  if (!session.client_reference_id) {
    // A paid checkout with no workspace ref can never be linked and bills
    // forever; no retry fixes it, so page a human.
    console.error(`${product} checkout link event missing client_reference_id`, {
      sessionId: session.id,
      eventId,
    });
    await alertOrphanedDeploySubscription({
      subscriptionId,
      sessionId: session.id,
      eventId,
      reason: "missing_client_reference_id",
      product,
    });
    return new Response("OK", { status: 200 });
  }

  const linkInput = {
    sessionId: session.id,
    expectedWorkspaceId: session.client_reference_id,
    audit: {
      actor: { type: "system" as const, id: "stripe" },
      location: "",
      userAgent: undefined,
    },
  };
  const result =
    productTag === "api"
      ? await linkApiSubscription(stripe, linkInput)
      : await linkDeploySubscription(stripe, linkInput);

  if (!result.ok) {
    console.error(`Failed to link ${product} checkout subscription`, {
      sessionId: session.id,
      eventId,
      reason: result.reason,
    });
    // Either way a paid subscription bills with no workspace attached and
    // no retry fixes it; page a human.
    if (
      result.reason === "subscription_conflict" ||
      result.reason === "workspace_not_found" ||
      result.reason === "invalid_api_product" ||
      result.reason === "no_api_plan" ||
      result.reason === "no_deploy_plan"
    ) {
      await alertOrphanedDeploySubscription({
        workspaceId: session.client_reference_id,
        subscriptionId,
        sessionId: session.id,
        eventId,
        reason: result.reason,
        product,
      });
    }
  }
  return new Response("OK", { status: 200 });
}

/**
 * Resolves the API plan item on a subscription and loads its price, customer,
 * and product. findApiItem skips any Deploy price the subscription might carry
 * (so a mixed subscription's tier is still derived from the API item, never a
 * Deploy line), falling back to items[0] only when Deploy billing is
 * unconfigured. Returns null when there is nothing to act on: no item, no
 * customer, or a price with no product/amount.
 */
async function resolveApiSubscriptionContext(
  stripe: Stripe,
  sub: Stripe.Subscription,
): Promise<{
  // Narrowed to a number here so callers do not have to re-check it; the guard
  // below rejects a price with no unit_amount.
  unitAmount: number;
  customer: Stripe.Customer | Stripe.DeletedCustomer;
  product: Stripe.Product;
} | null> {
  const config = await deployBillingConfig();
  const apiItem = findApiItem(config, sub.items?.data ?? []);
  if (!apiItem?.price?.id || !sub.customer) {
    return null;
  }

  const [price, customer] = await Promise.all([
    stripe.prices.retrieve(apiItem.price.id),
    stripe.customers.retrieve(typeof sub.customer === "string" ? sub.customer : sub.customer.id),
  ]);

  if (!price.product || price.unit_amount === null || price.unit_amount === undefined) {
    return null;
  }

  const product = await stripe.products.retrieve(
    typeof price.product === "string" ? price.product : price.product.id,
  );

  return { unitAmount: price.unit_amount, customer, product };
}

export const runtime = "nodejs";

export const POST = async (req: Request): Promise<Response> => {
  const signature = req.headers.get("stripe-signature");
  if (!signature) {
    console.error("Webhook signature validation failed: Missing stripe-signature header");
    return new Response("Webhook signature missing", { status: 400 });
  }

  const e = stripeEnv();

  if (!e) {
    console.error(
      "Stripe environment configuration is missing. Check that STRIPE_SECRET_KEY and other required Stripe environment variables are properly set.",
    );
    return new Response("Server configuration error", { status: 500 });
  }

  const stripeSecretKey = stripeEnv()?.STRIPE_SECRET_KEY;
  if (!stripeSecretKey) {
    console.error(
      "STRIPE_SECRET_KEY environment variable is not set. This is required for Stripe API operations.",
    );
    return new Response("Server configuration error", { status: 500 });
  }

  const stripe = new Stripe(stripeSecretKey, {
    apiVersion: "2026-06-24.dahlia",
    typescript: true,
  });

  let event: Stripe.Event;
  let requestBody: string;

  try {
    requestBody = await req.text();
  } catch (error) {
    console.error("Failed to read request body:", error);
    return new Response("Error", { status: 400 });
  }

  try {
    event = stripe.webhooks.constructEvent(requestBody, signature, e.STRIPE_WEBHOOK_SECRET);
  } catch (error) {
    console.error("Webhook signature validation failed:", error);
    return new Response("Error", { status: 400 });
  }
  switch (event.type) {
    case "customer.subscription.updated": {
      try {
        // The event snapshot only feeds the previous_attributes skip heuristics;
        // every DB write derives from a freshly retrieved subscription.
        const eventSub = event.data.object as Stripe.Subscription;

        // Resolve the subscription to its (workspace, product) with one
        // unique-index lookup on billing_subscriptions; the row's product
        // decides the branch, never the subscription's items.
        const subscription = await db.query.billingSubscriptions.findFirst({
          where: (table, { eq }) => eq(table.stripeSubscriptionId, eventSub.id),
          with: { workspace: { with: { billing: true } } },
        });
        const ws = subscription?.workspace ?? null;
        const billing = ws?.billing ?? null;
        if (!subscription || !ws || !billing || ws.deletedAtM !== null) {
          // Checkout can emit subscription.updated before
          // checkout.session.completed links the new subscription. Suppress
          // only this short, metadata-proven ordering window; the completed
          // event owns the durable link and orphan alert.
          const product = eventSub.metadata?.unkey_product;
          const isRecentCheckoutSubscription =
            (product === "api" || product === "compute") &&
            Boolean(eventSub.metadata?.workspace_id) &&
            eventSub.created * 1000 > Date.now() - 30 * 60 * 1000;
          if (isRecentCheckoutSubscription) {
            return new Response("OK", { status: 200 });
          }
          console.error("Workspace not found for subscription:", {
            subscriptionId: eventSub.id,
            eventId: event.id,
          });
          // A live subscription nothing points at keeps billing; page a human.
          await alertOrphanedDeploySubscription({
            subscriptionId: eventSub.id,
            eventId: event.id,
            reason: "workspace_not_found",
          });
          return new Response("OK", { status: 200 });
        }
        const column = subscription.product;

        // Stripe does not guarantee event ordering, so the snapshot can be stale;
        // derive every write from the live subscription. resource_missing means
        // the deleted handler owns it: ack.
        let sub: Stripe.Subscription;
        try {
          sub = await stripe.subscriptions.retrieve(eventSub.id);
        } catch (retrieveError) {
          if (
            retrieveError instanceof Stripe.errors.StripeError &&
            retrieveError.code === "resource_missing"
          ) {
            return new Response("OK", { status: 200 });
          }
          throw retrieveError;
        }

        // Read the event's previous_attributes once: the compute branch derives
        // its upgrade/downgrade/cancel alert from it, and the API branch below
        // uses it for its skip heuristics and up/downgrade copy.
        const previousAttributes = event.data.previous_attributes;

        // Deploy-matched: mirror the plan, announce the change, and stop. The
        // Deploy subscription never carries API tier/limit state, so there is
        // nothing else to reconcile. mirrorDeployPlan only writes when the plan
        // changed, so a renewal event does no DB write; the alert is derived
        // from the Stripe event (a DB plan diff is unreliable because the
        // dashboard mutations write the plan optimistically before this fires).
        if (column === "compute") {
          await deprovisionOnCancel(billing, sub);
          await mirrorDeployPlan(billing, ws.orgId, sub);
          const deployConfig = await deployBillingConfig();
          await sendComputeAlert(
            stripe,
            sub,
            computeUpdatedAlert(deployConfig, sub, previousAttributes),
            ws,
          );
          return new Response("OK", { status: 200 });
        }

        // Skip heuristics correlate the event snapshot with previous_attributes,
        // so they read eventSub (what the event reported), not the re-retrieved
        // subscription.
        // Skip database updates and notifications for automated billing renewals
        if (isAutomatedBillingRenewal(eventSub, previousAttributes)) {
          return new Response("OK", { status: 201 });
        }

        // Skip database updates and notifications for payment failure related updates
        // Payment failures are handled by the invoice.payment_failed webhook
        if (isPaymentFailureRelatedUpdate(eventSub, previousAttributes)) {
          return new Response("OK", { status: 201 });
        }

        // Skip database updates and notifications for payment recovery scenarios
        // Payment recoveries are handled by the invoice.payment_succeeded webhook
        const isRecovery = await isPaymentRecoveryUpdate(
          stripe,
          eventSub,
          previousAttributes,
          event,
        );
        if (isRecovery) {
          return new Response("OK", { status: 201 });
        }

        // Skip database updates and notifications for card/payment method updates only
        // These don't affect subscription pricing, limits, or other business logic
        if (isCardUpdateOnly(eventSub, previousAttributes)) {
          return new Response("OK", { status: 201 });
        }

        // Reconcile tier/limits from the API plan item. A missing item/price is
        // a degenerate API subscription with nothing to derive a tier from; ack
        // rather than guess.
        const apiContext = await resolveApiSubscriptionContext(stripe, sub);
        if (!apiContext) {
          return new Response("OK", { status: 200 });
        }
        const { unitAmount, customer, product } = apiContext;

        /**
         * In our case, when a user cancels their subscription, it's not in effect until the beginning of the next month.
         * So we get a subscription updated event, which we should handle accordingly.
         */
        if (sub.cancel_at) {
          // Alert only when an email exists, but return unconditionally: the tier
          // and limit update below is for active plan changes, never a cancelling
          // subscription.
          if (customer && !customer.deleted && customer.email) {
            await alertCustomerLifecycle({
              action: "cancelling",
              name: customer.name || "Unknown",
              email: customer.email,
              workspaceId: ws.id,
              workspaceName: ws.name,
              product: product.name,
              price: formatPrice(unitAmount),
              stripeCustomerId: customer.id,
              livemode: customer.livemode,
            });
          }
          return new Response("OK");
        }

        // Validate and parse Stripe limit metadata.
        const quotas = validateAndParseQuotas(product);
        if (!quotas.valid) {
          // Without valid quota metadata the tier sync is skipped while Stripe
          // bills the new plan; page a human to fix the product.
          console.error("Subscription update skipped: invalid product quota metadata", {
            productId: product.id,
            productName: product.name,
            subscriptionId: sub.id,
            eventId: event.id,
          });
          await alertInvalidProductQuotaMetadata({
            productId: product.id,
            productName: product.name,
            subscriptionId: sub.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        const { requestsPerMonth, logsRetentionDays, auditLogsRetentionDays } = quotas;

        /**
         * To make the updates more useful, we detect if they are downgrading or upgrading their subscription
         * We can then send a good or bad update based upon it.
         */
        let changeType = "updated";
        let previousTier: string | undefined;

        if (previousAttributes?.items?.data?.[0]?.price?.id) {
          try {
            const previousPrice = await stripe.prices.retrieve(
              previousAttributes.items.data[0].price.id,
            );

            if (previousPrice.product && previousPrice.unit_amount !== null) {
              const previousProduct = await stripe.products.retrieve(
                typeof previousPrice.product === "string"
                  ? previousPrice.product
                  : previousPrice.product.id,
              );

              previousTier = previousProduct.name;

              // Compare amounts to determine upgrade/downgrade
              const currentAmount = unitAmount;
              const previousAmount = previousPrice.unit_amount;

              if (currentAmount !== previousAmount && previousAmount !== null) {
                if (currentAmount > previousAmount) {
                  changeType = "upgraded";
                } else if (currentAmount < previousAmount) {
                  changeType = "downgraded";
                }
              }
            }
          } catch (error) {
            console.error("Error retrieving previous subscription details:", {
              error,
              eventId: event.id,
              subscriptionId: sub.id,
            });
          }
        }

        // The tRPC mutation clears these synchronously. Repeat it here because
        // Stripe is the source of truth and subscriptions can also be changed
        // outside the dashboard.
        const rateLimitReset =
          changeType === "upgraded" ? { apiRequestsCountMaxPerMinute: null } : {};

        // Update limits and workspace tier
        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaceBilling)
            .set({
              tier: product.name,
            })
            .where(eq(schema.workspaceBilling.workspaceId, ws.id));

          await setWorkspaceLimits(tx, {
            workspaceId: ws.id,
            plan: parseDeployPlan(billing.planOverride) ?? parseDeployPlan(billing.plan),
            preserveApiLimits: true,
            limitUpdate: {
              apiBillableOperationsCountMaxPerMonth: requestsPerMonth,
              logsRetentionDaysMax: logsRetentionDays,
              logsAuditRetentionDaysMax: auditLogsRetentionDays,
              teamEnabled: true,
              ...rateLimitReset,
            },
          });

          await insertAuditLogs(tx, {
            workspaceId: ws.id,
            actor: {
              type: "system",
              id: "stripe",
            },
            event: "workspace.update",
            description: `Subscription updated to ${product.name} plan.`,
            resources: [],
            context: {
              location: "",
              userAgent: undefined,
            },
          });
        });

        // Send notification for subscription update
        if (customer && !customer.deleted && customer.email) {
          await alertCustomerLifecycle({
            action:
              changeType === "upgraded"
                ? "upgrade"
                : changeType === "downgraded"
                  ? "downgrade"
                  : "update",
            name: customer.name || "Unknown",
            email: customer.email,
            workspaceId: ws.id,
            workspaceName: ws.name,
            product: product.name,
            previousProduct: previousTier,
            price: formatPrice(unitAmount),
            stripeCustomerId: customer.id,
            livemode: customer.livemode,
          });
        }
      } catch (error) {
        console.error("Subscription update webhook error:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });
        return new Response("Error", { status: 500 });
      }
      break;
    }
    case "customer.subscription.deleted": {
      try {
        const sub = event.data.object as Stripe.Subscription;

        // Resolve the ended subscription to its (workspace, product) with one
        // unique-index lookup on billing_subscriptions; the row's product
        // decides which product ended.
        const subscription = await db.query.billingSubscriptions.findFirst({
          where: (table, { eq }) => eq(table.stripeSubscriptionId, sub.id),
          with: { workspace: { with: { billing: true } } },
        });
        const ws = subscription?.workspace ?? null;
        const billing = ws?.billing ?? null;
        if (!subscription || !ws || !billing || ws.deletedAtM !== null) {
          console.error("Workspace not found for subscription:", {
            subscriptionId: sub.id,
            eventId: event.id,
          });
          // Only page when this is genuinely a lost billing link, not the second
          // delivery of a cancel we already processed. This handler deletes the
          // billing_subscriptions row it looks itself up by, and Stripe delivers
          // at-least-once, so a perfectly successful cancel used to page on
          // redelivery — as did any `updated` arriving after the `deleted`. That
          // trained the one alert meaning "a paid subscription is billing with no
          // workspace attached" to be ignored, which is exactly the signal a
          // Stripe-side cancel needs. A subscription Stripe reports as fully
          // ended has no ongoing billing, so silence is correct for it.
          if (subscriptionStillBilling(sub)) {
            await alertOrphanedDeploySubscription({
              subscriptionId: sub.id,
              eventId: event.id,
              reason: "workspace_not_found",
            });
          }
          return new Response("OK", { status: 200 });
        }
        const column = subscription.product;

        if (column === "compute") {
          // A Stripe-side cancel (e.g. the billing portal) removes the Deploy
          // subscription without going through the dashboard cancel flow, so
          // ctrl never tore down the workspace's Compute. If a Compute plan is
          // set, run the same keyed, idempotent teardown here: it stops running
          // compute and clears deploy_plan. Must run before the plan clear
          // below, since deprovisionCompute's idempotency guard keys on
          // deploy_plan still being set. Let a failure propagate so Stripe
          // retries; the teardown must not be dropped.
          if (billing.plan) {
            try {
              const ctrl = createCtrlClient(DeployService);
              await ctrl.deprovisionCompute({ workspaceId: ws.id });
            } catch (err) {
              console.error("Failed to deprovision Compute on Stripe-side cancel", {
                workspaceId: ws.id,
                subscriptionId: sub.id,
                error: err instanceof Error ? err.message : err,
              });
              throw err;
            }
          }

          // A paid API tier keeps team access after Compute ends.
          const keepsTeam = keepsTeamAfterDelete("compute", billing);

          // One transaction. Deleting the billing_subscriptions row is what the
          // retry lookup keys on, so a partial failure that committed it alone
          // would strand the workspace unfindable on redelivery. Making the
          // writes atomic means a retry re-finds the workspace.
          await db.transaction(async (tx) => {
            await tx
              .update(schema.workspaceBilling)
              .set({ plan: null })
              .where(eq(schema.workspaceBilling.workspaceId, ws.id));
            await deleteBillingSubscription(tx, { workspaceId: ws.id, product: "compute" });

            // Reset the Compute-owned ceilings even when a paid API plan remains;
            // in that case the API-owned limit fields and team access stay intact.
            await setWorkspaceLimits(tx, {
              workspaceId: ws.id,
              plan: null,
              preserveApiLimits: keepsTeam,
            });

            await insertAuditLogs(tx, {
              workspaceId: ws.id,
              actor: { type: "system", id: "stripe" },
              event: "workspace.update",
              description: "Cancelled Compute subscription.",
              resources: [],
              context: { location: "", userAgent: undefined },
            });
          });

          if (!keepsTeam) {
            await deactivateNonCreatorMemberships(ws.orgId);
          }

          // Notify that the Compute subscription ended. Best-effort: a failed
          // customer lookup or Slack post must not fail the webhook, whose
          // teardown has already committed.
          if (sub.customer) {
            try {
              const customer = await stripe.customers.retrieve(
                typeof sub.customer === "string" ? sub.customer : sub.customer.id,
              );
              if (!customer.deleted && customer.email) {
                await alertCustomerLifecycle({
                  action: "cancelled",
                  name: customer.name || "Unknown",
                  email: customer.email,
                  workspaceId: ws.id,
                  workspaceName: ws.name,
                  stripeCustomerId: customer.id,
                  livemode: customer.livemode,
                });
              }
            } catch (customerError) {
              console.error("Failed to retrieve customer for Compute cancellation alert:", {
                error: customerError,
                subscriptionId: sub.id,
                eventId: event.id,
              });
            }
          }
          break;
        }

        // API-matched: downgrade the API tier. Pro/Business Compute keeps team.
        const deployPlan = parseDeployPlan(billing.plan);
        const keepsTeam = keepsTeamAfterDelete("api", billing);

        // One transaction. Deleting the billing_subscriptions row is what the
        // retry lookup keys on, so if it committed alone and a later write
        // failed, the redelivered event could no longer find the workspace and
        // would strand paid limits on Free forever. Atomic writes mean a partial
        // failure rolls back the row delete too, so the retry re-finds the
        // workspace and completes the downgrade. Same shape as workspace creation.
        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaceBilling)
            .set({ tier: "Free" })
            .where(eq(schema.workspaceBilling.workspaceId, ws.id));
          await deleteBillingSubscription(tx, { workspaceId: ws.id, product: "api" });

          await setWorkspaceLimits(tx, {
            workspaceId: ws.id,
            plan: deployPlan,
            preserveApiLimits: false,
          });

          await insertAuditLogs(tx, {
            workspaceId: ws.id,
            actor: { type: "system", id: "stripe" },
            event: "workspace.update",
            description: keepsTeam
              ? "Cancelled API subscription; downgraded to Free (team retained via active Compute plan)."
              : "Cancelled API subscription.",
            resources: [],
            context: { location: "", userAgent: undefined },
          });
        });

        if (!keepsTeam) {
          // Free tier doesn't include team access — deactivate all members except
          // the original creator so lapsed subscriptions don't leave shared access on.
          await deactivateNonCreatorMemberships(ws.orgId);
        }

        // Send notification for subscription cancellation
        if (sub.customer) {
          try {
            const customer = await stripe.customers.retrieve(
              typeof sub.customer === "string" ? sub.customer : sub.customer.id,
            );

            if (customer && !customer.deleted && customer.email) {
              await alertCustomerLifecycle({
                action: "cancelled",
                name: customer.name || "Unknown",
                email: customer.email,
                workspaceId: ws.id,
                workspaceName: ws.name,
                stripeCustomerId: customer.id,
                livemode: customer.livemode,
              });
            }
          } catch (customerError) {
            console.error("Failed to retrieve customer for subscription cancellation alert:", {
              error: customerError,
              subscriptionId: sub.id,
              eventId: event.id,
            });
            // Continue without sending alert rather than failing
          }
        }
      } catch (error) {
        console.error("Subscription deletion webhook error:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });
        return new Response("Error", { status: 500 });
      }
      break;
    }
    case "customer.subscription.created": {
      /**
       * Subscription create + tier/limit writes happen inline in the createSubscription
       * tRPC mutation now. This webhook only sends the operational Slack alert so the
       * team is notified out-of-band; it deliberately does no DB writes.
       */
      try {
        const sub = event.data.object as Stripe.Subscription;

        // One unique-index lookup, then branch by the row's product. A created
        // event can race ahead of the tRPC/link write that inserts the
        // billing_subscriptions row, so a no-match is a best-effort no-op:
        // subscribeDeploy/linkDeploySubscription already write the plan inline,
        // and a later subscription.updated resyncs.
        const subscription = await db.query.billingSubscriptions.findFirst({
          where: (table, { eq }) => eq(table.stripeSubscriptionId, sub.id),
          with: { workspace: { with: { billing: true } } },
        });
        const ws = subscription?.workspace ?? null;
        const billing = ws?.billing ?? null;
        const column =
          subscription && ws && billing && ws.deletedAtM == null ? subscription.product : null;

        // Compute created: a created subscription carrying a recognized Compute
        // plan is a new Compute subscription. Detect it from the subscription's
        // own plan-fee item, not the billing_subscriptions row, because a created
        // event can race ahead of the checkout link that writes that row — gating
        // the alert on the row would drop it for the no-card checkout flow. Mirror
        // the plan when the row is already present (the link mirrors it
        // otherwise); announce the subscription either way. An API subscription
        // never carries Compute plan metadata, so this cannot misfire an API
        // create as a Compute alert.
        if (detectDeployPlan(sub)) {
          if (billing && ws) {
            await mirrorDeployPlan(billing, ws.orgId, sub);
          }
          await sendComputeAlert(stripe, sub, computeCreatedAlert(sub), ws);
          return new Response("OK");
        }

        // Not matched to a column yet (the create event raced ahead of the
        // tRPC/link write). No-op: the inline write set the plan and a later
        // subscription.updated resyncs. Only an API-matched create alerts, so we
        // never misfire a Compute create as an API subscription alert.
        if (column !== "api" || !ws) {
          return new Response("OK");
        }

        // API-matched: alert on the API plan item.
        const apiContext = await resolveApiSubscriptionContext(stripe, sub);
        if (!apiContext) {
          return new Response("OK");
        }
        const { unitAmount, customer, product } = apiContext;

        if (customer.deleted || !customer.email) {
          return new Response("OK");
        }

        await alertCustomerLifecycle({
          action: "signup",
          name: customer.name || "Unknown",
          email: customer.email,
          workspaceId: ws.id,
          workspaceName: ws.name,
          product: product.name,
          price: formatPrice(unitAmount),
          stripeCustomerId: customer.id,
          livemode: customer.livemode,
        });
        // Return rather than break so this case can never fall through into
        // invoice.payment_failed below; every other terminus in this case
        // returns too.
        return new Response("OK");
      } catch (error) {
        console.error("Subscription creation webhook error:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });
        return new Response("Error", { status: 500 });
      }
    }

    // Guaranteed server-side link for paid API and Compute flows: fires even if
    // the user never returns to /success. `completed` covers the immediate
    // (card) case; `async_payment_succeeded` covers delayed-notification
    // methods that reported unpaid at `completed` and only clear later. Both
    // route through the same idempotent linker, so a racing /success call or a
    // Stripe redelivery cannot double-write.
    case "checkout.session.completed":
    case "checkout.session.async_payment_succeeded": {
      try {
        const session = event.data.object as Stripe.Checkout.Session;
        return await linkCheckoutSession(stripe, session, event.id);
      } catch (error) {
        console.error("Checkout session link webhook error:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });
        return new Response("Error", { status: 500 });
      }
    }

    case "invoice.payment_failed": {
      try {
        const invoice = event.data.object as Stripe.Invoice;

        // Validate invoice data structure
        if (!invoice || typeof invoice !== "object") {
          console.error("Payment failed event received with invalid invoice data structure");
          return new Response("Invalid event data", { status: 400 });
        }

        // Extract customer information from the invoice
        if (!invoice.customer) {
          console.warn("Payment failed event received without customer information", {
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        let customer: Stripe.Customer | Stripe.DeletedCustomer;

        try {
          // Get customer details from Stripe with timeout handling
          customer = await stripe.customers.retrieve(
            typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id,
          );
        } catch (customerError) {
          console.error("Failed to retrieve customer for payment failure event:", {
            error: customerError,
            customerId:
              typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          // Continue processing without customer details rather than failing completely
          return new Response("OK", { status: 200 });
        }

        if (customer.deleted || !("email" in customer) || !customer.email) {
          return new Response("OK", { status: 200 });
        }

        // Extract payment failure details with validation
        const amount = invoice.amount_due || 0;

        // Validate amount
        if (amount < 0) {
          console.warn("Payment failed event with negative amount", {
            amount,
            invoiceId: invoice.id,
            eventId: event.id,
          });
        }

        try {
          // Send payment failure alert without triggering subscription updates
          const paymentCustomer = customer as Stripe.Customer;
          if (paymentCustomer.email) {
            await alertPaymentFailed({
              email: paymentCustomer.email,
              name: paymentCustomer.name || "Unknown",
              amount,
              stripeCustomerId: paymentCustomer.id,
              livemode: paymentCustomer.livemode,
            });
          }
        } catch (alertError) {
          console.error("Failed to send payment failure alert:", {
            error: alertError,
            customerEmail: (customer as Stripe.Customer).email,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          // Don't fail the webhook if alert fails - return success to prevent retries
          return new Response("Alert failed but event processed", { status: 200 });
        }

        // Return success immediately to prevent fall-through to other webhook handlers
        return new Response("OK", { status: 200 });
      } catch (error) {
        console.error("Error processing payment failure webhook:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });

        // Return 200 to prevent Stripe from retrying, but log the error
        // This ensures payment processing errors don't affect other webhook types
        return new Response("Error processing payment failure", { status: 200 });
      }
    }

    // invoice.paid also covers out-of-band and fully-credit-covered
    // settlements. The grant is idempotent per invoice, so double-firing
    // no-ops; recovery alerts stay on payment_succeeded.
    case "invoice.paid": {
      try {
        const invoice = event.data.object as Stripe.Invoice;

        if (!invoice || typeof invoice !== "object" || !invoice.customer) {
          return new Response("OK", { status: 200 });
        }

        try {
          const grant = await grantDeployCreditsForInvoice(stripe, invoice);
          if (grant.granted) {
            console.info("Granted Deploy usage credits", {
              invoiceId: invoice.id,
              grantId: grant.grantId,
              amountCents: grant.amountCents,
            });
          } else {
            console.info("Did not grant Deploy usage credits", {
              invoiceId: invoice.id,
              reason: grant.reason,
            });
          }
        } catch (grantError) {
          console.error("Failed to grant Deploy usage credits:", {
            error: grantError,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("Error granting Deploy credits", { status: 500 });
        }

        return new Response("OK", { status: 200 });
      } catch (error) {
        console.error("Error processing invoice.paid webhook:", {
          error:
            error instanceof Error
              ? { message: error.message, stack: error.stack, name: error.name }
              : error,
          eventId: event.id,
          eventType: event.type,
        });
        // The only work here is the idempotent grant, so a retry is safe.
        return new Response("Error processing invoice.paid", { status: 500 });
      }
    }

    case "invoice.payment_succeeded": {
      try {
        const invoice = event.data.object as Stripe.Invoice;

        // Validate invoice data structure
        if (!invoice || typeof invoice !== "object") {
          console.error("Payment success event received with invalid invoice data structure");
          return new Response("Invalid event data", { status: 400 });
        }

        // Extract customer information from the invoice
        if (!invoice.customer) {
          console.warn("Payment success event received without customer information", {
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("OK", { status: 200 });
        }

        // A paid invoice carrying a Deploy plan-fee entitles the workspace to
        // usage credits equal to the fee. This must run before the alert
        // logic below, whose early returns (deleted customer, no email) must
        // not skip the grant. Failure returns 500 so Stripe retries; the
        // grant is idempotent per invoice, so retries cannot double-grant.
        try {
          const grant = await grantDeployCreditsForInvoice(stripe, invoice);
          if (grant.granted) {
            console.info("Granted Deploy usage credits", {
              invoiceId: invoice.id,
              grantId: grant.grantId,
              amountCents: grant.amountCents,
            });
          } else {
            // No credits is usually a deliberate skip: no Deploy plan-fee
            // line, period already closed, already granted, or non-positive
            // net. Log the reason so the skip is explained.
            console.info("Did not grant Deploy usage credits", {
              invoiceId: invoice.id,
              reason: grant.reason,
            });
          }
        } catch (grantError) {
          console.error("Failed to grant Deploy usage credits:", {
            error: grantError,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          return new Response("Error granting Deploy credits", { status: 500 });
        }

        let customer: Stripe.Customer | Stripe.DeletedCustomer;

        try {
          // Get customer details from Stripe with timeout handling
          customer = await stripe.customers.retrieve(
            typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id,
          );
        } catch (customerError) {
          console.error("Failed to retrieve customer for payment success event:", {
            error: customerError,
            customerId:
              typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id,
            invoiceId: invoice.id,
            eventId: event.id,
          });
          // Continue processing without customer details rather than failing completely
          return new Response("OK", { status: 200 });
        }

        if (customer.deleted || !("email" in customer) || !customer.email) {
          return new Response("OK", { status: 200 });
        }

        let isRecovery = false;

        try {
          // Use recovery detection logic to determine if success follows failure
          isRecovery = await isPaymentRecovery(stripe, event);
        } catch (recoveryError) {
          console.error("Failed to determine payment recovery status:", {
            error: recoveryError,
            invoiceId: invoice.id,
            eventId: event.id,
            customerEmail: customer.email,
          });
          // Assume not a recovery if detection fails to avoid false positives
          isRecovery = false;
        }

        // Send recovery alert only when appropriate (after previous failures)
        if (isRecovery) {
          const amount = invoice.amount_paid || 0;

          // Validate amount
          if (amount < 0) {
            console.warn("Payment success event with negative amount", {
              amount,
              invoiceId: invoice.id,
              eventId: event.id,
            });
          }

          const paymentCustomer = customer as Stripe.Customer;
          if (paymentCustomer.email) {
            try {
              await alertPaymentRecovered({
                email: paymentCustomer.email,
                name: paymentCustomer.name || "Unknown",
                amount,
                stripeCustomerId: paymentCustomer.id,
                livemode: paymentCustomer.livemode,
              });
            } catch (alertError) {
              console.error("Failed to send payment recovery alert:", {
                error: alertError,
                invoiceId: invoice.id,
                eventId: event.id,
              });
              // Don't fail the webhook if alert fails - return success to prevent retries
              return new Response("Alert failed but event processed", { status: 200 });
            }
          }
        }

        // Return success immediately to prevent fall-through to other webhook handlers
        return new Response("OK", { status: 200 });
      } catch (error) {
        console.error("Error processing payment success webhook:", {
          error:
            error instanceof Error
              ? {
                  message: error.message,
                  stack: error.stack,
                  name: error.name,
                }
              : error,
          eventId: event.id,
          eventType: event.type,
        });

        // Return 200 to prevent Stripe from retrying, but log the error
        // This ensures payment processing errors don't affect other webhook types
        return new Response("Error processing payment success", { status: 200 });
      }
    }

    default:
      break;
  }
  return new Response("OK");
};
