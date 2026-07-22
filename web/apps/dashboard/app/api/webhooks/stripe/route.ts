import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { insertAuditLogs } from "@/lib/audit";
import { auth } from "@/lib/auth/server";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { formatPrice } from "@/lib/fmt";
import { freeTierQuotas } from "@/lib/quotas";
import {
  deployBillingConfig,
  deployBillingConfigured,
  findApiItem,
} from "@/lib/stripe/deployBilling";
import { grantDeployCreditsForInvoice } from "@/lib/stripe/deployCredits";
import { detectDeployPlan } from "@/lib/stripe/deployPlan";
import { linkDeploySubscription } from "@/lib/stripe/linkDeploySubscription";
import { isPaymentRecovery, isPaymentRecoveryUpdate } from "@/lib/stripe/paymentUtils";
import { validateAndParseQuotas } from "@/lib/stripe/productUtils";
import {
  getApiCancelSchedule,
  isAutomatedBillingRenewal,
  isCardUpdateOnly,
  isPaymentFailureRelatedUpdate,
} from "@/lib/stripe/subscriptionUtils";
import {
  alertInvalidProductQuotaMetadata,
  alertIsCancellingSubscription,
  alertOrphanedDeploySubscription,
  alertPaymentFailed,
  alertPaymentRecovered,
  alertSubscriptionCancelled,
  alertSubscriptionCreation,
  alertSubscriptionUpdate,
} from "@/lib/utils/slackAlerts";
import Stripe from "stripe";

/**
 * Mirrors a subscription's Deploy plan onto its workspace row, writing only when
 * it changed so the common renewal case (plan unchanged) does no DB write.
 * Stripe stays the source of truth; workspaces.deploy_plan is the cache the
 * deploy gate and dashboard read without a Stripe call in the hot path.
 */
async function mirrorDeployPlan(
  billing: { workspaceId: string; plan: string | null },
  sub: Stripe.Subscription,
): Promise<void> {
  const plan = detectDeployPlan(sub);
  const changed = plan !== billing.plan;
  if (changed) {
    await db
      .update(schema.workspaceBilling)
      .set({ plan })
      .where(eq(schema.workspaceBilling.workspaceId, billing.workspaceId));
  }
}

/**
 * Completes a scheduled API-plan cancellation: when cancelSubscription phases
 * the API items off a mixed subscription, the phase boundary arrives as a
 * subscription.updated whose items no longer include an API plan. Mirror the
 * customer.subscription.deleted downgrade (tier, quotas) but keep the
 * subscription linked — Compute continues to bill on it.
 *
 * Team access follows ANY active paid subscription, so while the surviving
 * Compute plan is live the members stay active and the team quota stays on;
 * they only go when the last paid plan does (the whole-subscription deleted
 * handler). Everything else with no API item is a no-op: Compute-only
 * subscriptions are already on the Free tier, and an unmarked schedule (or
 * none) means the item disappeared some other way we must not react to.
 */
async function downgradeAfterScheduledApiCancel(
  stripe: Stripe,
  ws: { id: string; orgId: string; tier: string | null },
  sub: Stripe.Subscription,
): Promise<Response> {
  if (ws.tier === "Free") {
    // Compute-only subscriptions and webhook redeliveries land here.
    return new Response("OK", { status: 200 });
  }
  const schedule = await getApiCancelSchedule(stripe, sub);
  if (!schedule) {
    return new Response("OK", { status: 200 });
  }

  // Read the surviving Compute plan off the event's subscription rather than
  // the workspace row: mirrorDeployPlan already reconciled the DB, but the
  // in-memory row predates it.
  const keepsTeam = detectDeployPlan(sub) !== null;

  // When a Compute plan survives, reset only the API-scoped quota fields:
  // the Compute resource ceilings belong to the surviving plan, so spreading
  // all of freeTierQuotas would clamp them to Free the moment Compute tiers
  // diverge from the Free defaults.
  const apiFreeQuotas = {
    requestsPerMonth: freeTierQuotas.requestsPerMonth,
    logsRetentionDays: freeTierQuotas.logsRetentionDays,
    auditLogsRetentionDays: freeTierQuotas.auditLogsRetentionDays,
    ratelimitApiLimit: freeTierQuotas.ratelimitApiLimit,
    ratelimitApiDuration: freeTierQuotas.ratelimitApiDuration,
  };
  const downgradedQuotas = keepsTeam ? { ...apiFreeQuotas, team: true } : freeTierQuotas;

  // The redelivery guard returns early on tier === "Free", so the tier
  // write must come last or a partial failure would never be retried.
  await db
    .insert(schema.quotas)
    .values({
      workspaceId: ws.id,
      ...downgradedQuotas,
    })
    .onDuplicateKeyUpdate({
      set: downgradedQuotas,
    });

  await insertAuditLogs(db, {
    workspaceId: ws.id,
    actor: { type: "system", id: "stripe" },
    event: "workspace.update",
    description: keepsTeam
      ? "Scheduled API plan cancellation took effect; downgraded to Free (team retained via active Compute plan)."
      : "Scheduled API plan cancellation took effect; downgraded to Free.",
    resources: [],
    context: { location: "", userAgent: undefined },
  });

  if (!keepsTeam) {
    // Free tier doesn't include team access — deactivate all members except
    // the original creator, same as a whole-subscription cancellation.
    await deactivateNonCreatorMemberships(ws.orgId);
  }

  await db
    .update(schema.workspaceBilling)
    .set({ tier: "Free" })
    .where(eq(schema.workspaceBilling.workspaceId, ws.id));

  return new Response("OK", { status: 200 });
}

/**
 * Links a subscription-mode Compute checkout to its workspace via the shared
 * linker, shared by the `checkout.session.completed` and
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
async function linkComputeCheckoutSession(
  stripe: Stripe,
  session: Stripe.Checkout.Session,
  eventId: string,
): Promise<Response> {
  if (session.mode !== "subscription" || !session.subscription) {
    return new Response("OK", { status: 200 });
  }
  const subscriptionId =
    typeof session.subscription === "string" ? session.subscription : session.subscription.id;
  if (!session.client_reference_id) {
    // A paid checkout with no workspace ref can never be linked and bills
    // forever; no retry fixes it, so page a human.
    console.error("Compute checkout link event missing client_reference_id", {
      sessionId: session.id,
      eventId,
    });
    await alertOrphanedDeploySubscription({
      subscriptionId,
      sessionId: session.id,
      eventId,
      reason: "missing_client_reference_id",
    });
    return new Response("OK", { status: 200 });
  }

  const result = await linkDeploySubscription(stripe, {
    sessionId: session.id,
    expectedWorkspaceId: session.client_reference_id,
    audit: { actor: { type: "system", id: "stripe" }, location: "", userAgent: undefined },
  });

  if (!result.ok) {
    console.error("Failed to link Compute checkout subscription", {
      sessionId: session.id,
      eventId,
      reason: result.reason,
    });
    // Either way a paid subscription bills with no workspace attached and
    // no retry fixes it; page a human.
    if (result.reason === "subscription_conflict" || result.reason === "workspace_not_found") {
      await alertOrphanedDeploySubscription({
        workspaceId: session.client_reference_id,
        subscriptionId,
        sessionId: session.id,
        eventId,
        reason: result.reason,
      });
    }
  }
  return new Response("OK", { status: 200 });
}

/**
 * Resolves the API plan item on a subscription and loads its price, customer,
 * and product. Anchors on findApiItem rather than items[0] so a mixed
 * (API + Compute) subscription resolves the API product, not a Compute item.
 * Returns null when there is nothing to act on: no API item (e.g. a
 * Compute-only subscription), no customer, or a price with no product/amount.
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
  // Config resolution failure would let findApiItem misclassify a Compute
  // item and ack a scheduled downgrade away; throw so Stripe retries. An
  // unconfigured deployment keeps the legacy items[0] behavior.
  if (!config && deployBillingConfigured()) {
    throw new Error("Deploy billing configured but unresolved; retrying webhook");
  }

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

/**
 * Deactivates every active membership in `orgId` except the earliest one (the original
 * creator). Determining the creator from membership createdAt avoids storing extra DB
 * state. Errors per-membership are logged but don't fail the webhook — partial revocation
 * is preferable to leaving the workspace stuck in an inconsistent paid state.
 */
async function deactivateNonCreatorMemberships(orgId: string): Promise<void> {
  let memberships: Awaited<ReturnType<typeof auth.getOrganizationMemberList>>;
  try {
    memberships = await auth.getOrganizationMemberList(orgId);
  } catch (err) {
    console.error("Failed to list memberships for deactivation:", { orgId, error: err });
    return;
  }

  if (memberships.data.length <= 1) {
    return;
  }

  const sorted = [...memberships.data].sort((a, b) => a.createdAt.localeCompare(b.createdAt));
  const [, ...nonCreators] = sorted;

  await Promise.all(
    nonCreators.map(async (member) => {
      try {
        await auth.deactivateMembership(member.id, orgId);
      } catch (err) {
        console.error("Failed to deactivate membership:", {
          orgId,
          membershipId: member.id,
          error: err,
        });
      }
    }),
  );
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

        const billing = await db.query.workspaceBilling.findFirst({
          where: (table, { eq }) => eq(table.stripeSubscriptionId, eventSub.id),
          with: { workspace: true },
        });
        const ws = billing?.workspace ?? null;
        if (!billing || !ws || ws.deletedAtM !== null) {
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

        // Sync before the skip-paths below: a plan add/change/remove must be
        // mirrored even when the rest of the update is a no-op (renewal, card
        // update) or bails on the API-tier quota validation, which a Deploy-only
        // subscription does not satisfy.
        await mirrorDeployPlan(billing, sub);

        const previousAttributes = event.data.previous_attributes;

        // Skip heuristics correlate with previous_attributes, so they read eventSub.
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
        // These don't affect subscription pricing, quotas, or other business logic
        if (isCardUpdateOnly(eventSub, previousAttributes)) {
          return new Response("OK", { status: 201 });
        }

        // Reconcile tier/quotas from the API plan item (the Deploy signal is
        // mirrored above). No API item either means a Compute-only
        // subscription (no-op) or a scheduled API-plan cancellation whose
        // phase boundary just hit — downgradeAfterScheduledApiCancel tells
        // them apart and downgrades the workspace for the latter.
        const apiContext = await resolveApiSubscriptionContext(stripe, sub);
        if (!apiContext) {
          return await downgradeAfterScheduledApiCancel(
            stripe,
            { id: ws.id, orgId: ws.orgId, tier: billing.tier },
            sub,
          );
        }
        const { unitAmount, customer, product } = apiContext;

        /**
         * In our case, when a user cancels their subscription, it's not in effect until the beginning of the next month.
         * So we get a subscription updated event, which we should handle accordingly.
         */
        if (sub.cancel_at) {
          // Alert only when an email exists, but return unconditionally: the tier
          // and quota update below is for active plan changes, never a cancelling
          // subscription.
          if (customer && !customer.deleted && customer.email) {
            const formattedPrice = formatPrice(unitAmount);
            await alertIsCancellingSubscription(
              product.name,
              formattedPrice,
              customer.email,
              customer.name || "Unknown",
            );
          }
          return new Response("OK");
        }

        // Validate and parse quotas
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

        // Update quotas and workspace tier
        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaceBilling)
            .set({
              tier: product.name,
            })
            .where(eq(schema.workspaceBilling.workspaceId, ws.id));

          await tx
            .insert(schema.quotas)
            .values({
              workspaceId: ws.id,
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

        // Send notification for subscription update
        if (customer && !customer.deleted && customer.email) {
          const formattedPrice = formatPrice(unitAmount);

          await alertSubscriptionUpdate(
            product.name,
            formattedPrice,
            customer.email,
            customer.name || "Unknown",
            changeType,
            previousTier,
          );
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

        const billing = await db.query.workspaceBilling.findFirst({
          where: (table, { eq }) => eq(table.stripeSubscriptionId, sub.id),
          with: { workspace: true },
        });
        const ws = billing?.workspace ?? null;
        if (!billing || !ws || ws.deletedAtM !== null) {
          console.error("Workspace not found for subscription:", {
            subscriptionId: sub.id,
            eventId: event.id,
          });
          // No ongoing billing on a terminated subscription, but the lost billing
          // link needs a human to reconcile.
          await alertOrphanedDeploySubscription({
            subscriptionId: sub.id,
            eventId: event.id,
            reason: "workspace_not_found",
          });
          return new Response("OK", { status: 200 });
        }

        // A Stripe-side cancel (e.g. the billing portal) removes the subscription
        // without going through the dashboard cancel flow, so ctrl never tore down
        // the workspace's Compute. If this workspace had a Compute plan, run the
        // same keyed, idempotent teardown here: it stops running compute and clears
        // deploy_plan. Must run before the column clear below, since
        // deprovisionCompute's idempotency guard keys on deploy_plan still being set.
        // Let a failure propagate so Stripe retries; the teardown must not be dropped.
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

        // One transaction: the link clear is the retry lookup key, so committing
        // it alone would strand the redelivered event. Same shape as workspace
        // creation.
        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaceBilling)
            .set({
              stripeSubscriptionId: null,
              tier: "Free",
              // The subscription is gone, so the Deploy plan goes with it.
              plan: null,
            })
            .where(eq(schema.workspaceBilling.workspaceId, ws.id));

          await tx
            .insert(schema.quotas)
            .values({
              workspaceId: ws.id,
              ...freeTierQuotas,
            })
            .onDuplicateKeyUpdate({
              set: freeTierQuotas,
            });

          await insertAuditLogs(tx, {
            workspaceId: ws.id,
            actor: {
              type: "system",
              id: "stripe",
            },
            event: "workspace.update",
            description: "Cancelled subscription.",
            resources: [],
            context: {
              location: "",
              userAgent: undefined,
            },
          });
        });

        // Free tier doesn't include team access — deactivate all members except the
        // original creator so lapsed subscriptions don't leave shared access enabled.
        await deactivateNonCreatorMemberships(ws.orgId);

        // Send notification for subscription cancellation
        if (sub.customer) {
          try {
            const customer = await stripe.customers.retrieve(
              typeof sub.customer === "string" ? sub.customer : sub.customer.id,
            );

            if (customer && !customer.deleted && customer.email) {
              await alertSubscriptionCancelled(customer.email, customer.name || "Unknown");
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
       * Subscription create + tier/quota writes happen inline in the createSubscription
       * tRPC mutation now. This webhook only sends the operational Slack alert so the
       * team is notified out-of-band; it deliberately does no DB writes.
       */
      try {
        const sub = event.data.object as Stripe.Subscription;

        // Mirror the Deploy plan signal when the new subscription already carries
        // a Deploy plan-fee item (the free-tier path creates a subscription with
        // Deploy items as its initial set). Best-effort: if the workspace row is
        // not linked to this subscription yet, a later subscription.updated syncs
        // it. Done before the alert-only logic below so an early return can't skip
        // it.
        const billingForDeploy = await db.query.workspaceBilling.findFirst({
          where: (table, { eq }) => eq(table.stripeSubscriptionId, sub.id),
          with: { workspace: true },
        });
        if (billingForDeploy && billingForDeploy.workspace?.deletedAtM == null) {
          await mirrorDeployPlan(billingForDeploy, sub);
        }

        // Alert on the API plan item (the Deploy signal is mirrored above).
        // Nothing to alert on for a Compute-only subscription.
        const apiContext = await resolveApiSubscriptionContext(stripe, sub);
        if (!apiContext) {
          return new Response("OK");
        }
        const { unitAmount, customer, product } = apiContext;

        if (customer.deleted || !customer.email) {
          return new Response("OK");
        }

        const formattedPrice = formatPrice(unitAmount);

        await alertSubscriptionCreation(
          product.name,
          formattedPrice,
          customer.email,
          customer.name || "Unknown",
        );
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

    // Guaranteed server-side link for the no-card Compute flow: fires even if
    // the user never returns to /success. `completed` covers the immediate
    // (card) case; `async_payment_succeeded` covers delayed-notification
    // methods that reported unpaid at `completed` and only clear later. Both
    // route through the same idempotent linker, so a racing /success call or a
    // Stripe redelivery cannot double-write.
    case "checkout.session.completed":
    case "checkout.session.async_payment_succeeded": {
      try {
        const session = event.data.object as Stripe.Checkout.Session;
        return await linkComputeCheckoutSession(stripe, session, event.id);
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
        const currency = invoice.currency || "usd";

        // Validate amount and currency
        if (amount < 0) {
          console.warn("Payment failed event with negative amount", {
            amount,
            invoiceId: invoice.id,
            eventId: event.id,
          });
        }

        try {
          // Send payment failure alert without triggering subscription updates
          const customerEmail = (customer as Stripe.Customer).email;
          if (customerEmail) {
            await alertPaymentFailed(
              customerEmail,
              (customer as Stripe.Customer).name || "Unknown",
              amount,
              currency,
            );
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
          const currency = invoice.currency || "usd";

          // Validate amount and currency
          if (amount < 0) {
            console.warn("Payment success event with negative amount", {
              amount,
              invoiceId: invoice.id,
              eventId: event.id,
            });
          }

          const customerEmail = (customer as Stripe.Customer).email;
          if (customerEmail) {
            try {
              await alertPaymentRecovered(
                customerEmail,
                (customer as Stripe.Customer).name || "Unknown",
                amount,
                currency,
              );
            } catch (alertError) {
              console.error("Failed to send payment recovery alert:", {
                error: alertError,
                customerEmail,
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
