import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { insertAuditLogs } from "@/lib/audit";
import { auth } from "@/lib/auth/server";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { formatPrice } from "@/lib/fmt";
import { freeTierQuotas } from "@/lib/quotas";
import { grantDeployCreditsForInvoice } from "@/lib/stripe/deployCredits";
import { detectDeployPlan } from "@/lib/stripe/deployPlan";
import { linkDeploySubscription } from "@/lib/stripe/linkDeploySubscription";
import { isPaymentRecovery, isPaymentRecoveryUpdate } from "@/lib/stripe/paymentUtils";
import { validateAndParseQuotas } from "@/lib/stripe/productUtils";
import {
  isAutomatedBillingRenewal,
  isCardUpdateOnly,
  isPaymentFailureRelatedUpdate,
} from "@/lib/stripe/subscriptionUtils";
import { keepsTeamAfterDelete, matchSubscriptionColumn } from "@/lib/stripe/webhookRouting";
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
 * and product. The API subscription carries only the API plan item now (the
 * split gave Deploy its own subscription), so items[0] is the API item.
 * Returns null when there is nothing to act on: no item, no customer, or a
 * price with no product/amount.
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
  const apiItem = sub.items?.data?.[0];
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

        // One OR lookup on both subscription-id columns; the column that
        // matched decides the product, never the subscription's items.
        const billing = await db.query.workspaceBilling.findFirst({
          where: (table, { eq, or }) =>
            or(
              eq(table.stripeSubscriptionId, eventSub.id),
              eq(table.stripeDeploySubscriptionId, eventSub.id),
            ),
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
        const column = matchSubscriptionColumn(billing, eventSub.id);

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

        // Deploy-matched: mirror the plan and stop. The Deploy subscription
        // never carries API tier/quota state, so there is nothing else to
        // reconcile. mirrorDeployPlan only writes when the plan changed, so a
        // renewal event is a no-op.
        if (column === "deploy") {
          await mirrorDeployPlan(billing, sub);
          return new Response("OK", { status: 200 });
        }

        // API-matched from here: reconcile tier/quotas from the API plan item.
        const previousAttributes = event.data.previous_attributes;

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
        // These don't affect subscription pricing, quotas, or other business logic
        if (isCardUpdateOnly(eventSub, previousAttributes)) {
          return new Response("OK", { status: 201 });
        }

        // Reconcile tier/quotas from the API plan item. A missing item/price is
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

        // One OR lookup on both subscription-id columns; the matched column
        // decides which product ended.
        const billing = await db.query.workspaceBilling.findFirst({
          where: (table, { eq, or }) =>
            or(
              eq(table.stripeSubscriptionId, sub.id),
              eq(table.stripeDeploySubscriptionId, sub.id),
            ),
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
        const column = matchSubscriptionColumn(billing, sub.id);

        if (column === "deploy") {
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

          // Team follows any live paid product, so a paid API tier keeps it.
          const keepsTeam = keepsTeamAfterDelete("deploy", billing);

          // One transaction. The link clear (stripeDeploySubscriptionId = null)
          // is what the retry lookup keys on, so a partial failure that
          // committed it alone would strand the row unfindable on redelivery.
          // Making the writes atomic means a retry re-finds the workspace.
          await db.transaction(async (tx) => {
            await tx
              .update(schema.workspaceBilling)
              .set({ stripeDeploySubscriptionId: null, plan: null })
              .where(eq(schema.workspaceBilling.workspaceId, ws.id));

            // Only reset quotas when nothing paid remains; a paid API tier keeps
            // its own quotas, so leave them untouched.
            if (!keepsTeam) {
              await tx
                .insert(schema.quotas)
                .values({ workspaceId: ws.id, ...freeTierQuotas })
                .onDuplicateKeyUpdate({ set: freeTierQuotas });
            }

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
          break;
        }

        // API-matched: downgrade the API tier. An active Deploy plan keeps team.
        const keepsTeam = keepsTeamAfterDelete("api", billing);

        // When a Compute plan survives, reset only the API-scoped quota fields:
        // the Compute resource ceilings belong to the surviving plan, so
        // spreading all of freeTierQuotas would clamp them to Free the moment
        // Compute tiers diverge from the Free defaults.
        const apiFreeQuotas = {
          requestsPerMonth: freeTierQuotas.requestsPerMonth,
          logsRetentionDays: freeTierQuotas.logsRetentionDays,
          auditLogsRetentionDays: freeTierQuotas.auditLogsRetentionDays,
          ratelimitApiLimit: freeTierQuotas.ratelimitApiLimit,
          ratelimitApiDuration: freeTierQuotas.ratelimitApiDuration,
        };
        const downgradedQuotas = keepsTeam ? { ...apiFreeQuotas, team: true } : freeTierQuotas;

        // One transaction. The link clear (stripeSubscriptionId = null) is what
        // the retry lookup keys on, so if it committed alone and a later write
        // failed, the redelivered event could no longer find the workspace and
        // would strand paid quotas on Free forever. Atomic writes mean a partial
        // failure rolls back the link clear too, so the retry re-finds the
        // workspace and completes the downgrade. Same shape as workspace creation.
        await db.transaction(async (tx) => {
          await tx
            .update(schema.workspaceBilling)
            .set({ stripeSubscriptionId: null, tier: "Free" })
            .where(eq(schema.workspaceBilling.workspaceId, ws.id));

          await tx
            .insert(schema.quotas)
            .values({ workspaceId: ws.id, ...downgradedQuotas })
            .onDuplicateKeyUpdate({ set: downgradedQuotas });

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

        // One OR lookup, then branch by the matched column. A created event can
        // race ahead of the tRPC/link write that sets the column, so a no-match
        // is a best-effort no-op: subscribeDeploy/linkDeploySubscription already
        // write the plan inline, and a later subscription.updated resyncs.
        const billing = await db.query.workspaceBilling.findFirst({
          where: (table, { eq, or }) =>
            or(
              eq(table.stripeSubscriptionId, sub.id),
              eq(table.stripeDeploySubscriptionId, sub.id),
            ),
          with: { workspace: true },
        });
        const column =
          billing && billing.workspace?.deletedAtM == null
            ? matchSubscriptionColumn(billing, sub.id)
            : null;

        // Deploy-matched: mirror the plan and stop; no alert for Compute.
        if (column === "deploy" && billing) {
          await mirrorDeployPlan(billing, sub);
          return new Response("OK");
        }

        // Not matched to a column yet (the create event raced ahead of the
        // tRPC/link write). No-op: the inline write set the plan and a later
        // subscription.updated resyncs. Only an API-matched create alerts, so we
        // never misfire a Compute create as an API subscription alert.
        if (column !== "api") {
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
