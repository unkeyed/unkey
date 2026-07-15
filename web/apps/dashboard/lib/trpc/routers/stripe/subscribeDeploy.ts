import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import {
  deployBillingConfig,
  deploySubscriptionItems,
  findDeployItems,
} from "@/lib/stripe/deployBilling";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { isDeadSubscription } from "@/lib/stripe/subscriptionUtils";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";
import { assertSubscriptionAttachable } from "./subscriptionGuards";

/**
 * Subscribes the workspace to an Unkey Deploy plan: attaches the plan-fee price
 * for the chosen tier plus the shared metered prices to the workspace's Stripe
 * subscription, creating the subscription first if the workspace is on the free
 * tier with no subscription yet.
 *
 * Writes workspaces.deploy_plan optimistically so the UI reflects the new plan
 * immediately. Stripe stays source of truth: the resulting customer.subscription.*
 * webhook reconciles the column, and since it derives the same value from the
 * subscription we just mutated, that reconciliation is a no-op. The
 * no-subscription path also writes stripeSubscriptionId so the webhook can find
 * this workspace.
 */
export const subscribeDeploy = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(
    z.object({
      plan: z.enum(DEPLOY_PLANS),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const config = await deployBillingConfig();
    if (!config) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Compute billing is not configured.",
      });
    }

    if (!ctx.workspace.stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Add a payment method before subscribing to a Compute plan.",
      });
    }

    if (ctx.workspace.deployPlan) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace already has a Compute plan. Use change instead.",
      });
    }

    const stripe = getStripeClient();
    const items = deploySubscriptionItems(config, input.plan);

    // Columns to write once the Stripe mutation below succeeds, set in whichever
    // branch runs. Written optimistically so the UI reflects the new plan now;
    // the customer.subscription.* webhook reconciles them (a no-op, since it
    // derives the same value from the subscription we just mutated).
    let workspaceUpdate: { deployPlan: string; stripeSubscriptionId?: string };

    // The recorded subscription can be a corpse (cancelDeploy cancels a
    // Compute-only subscription outright, and the deleted-webhook that clears
    // the column may not have landed yet). A dead one still carries its old
    // Compute items, so without this check a mid-month resubscribe dies on the
    // "already has Compute items" guard below. Treat it as absent and create a
    // fresh subscription instead.
    let existingSub: Stripe.Subscription | null = null;
    if (ctx.workspace.stripeSubscriptionId) {
      const recorded = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);
      if (!isDeadSubscription(recorded)) {
        existingSub = recorded;
      }
    }

    if (existingSub) {
      // Existing subscription (e.g. a paid API plan): append the Deploy items.
      // Items not listed here are left untouched, so API items are preserved.
      const sub = existingSub;

      if (findDeployItems(config, sub.items.data).length > 0) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "Workspace already has Compute items on its subscription.",
        });
      }

      // Deploy items only attach to a subscription that will actually keep
      // billing them; reject anything else here instead of letting Stripe
      // attach to a subscription that ends at the period roll.
      assertSubscriptionAttachable(sub);

      try {
        await stripe.subscriptions.update(sub.id, {
          items,
          proration_behavior: "always_invoice",
          payment_behavior: "error_if_incomplete",
        });
      } catch (err) {
        throw toBillingError(err);
      }

      workspaceUpdate = { deployPlan: input.plan };
    } else {
      // Free tier: create a subscription whose initial items are the Deploy set.
      // error_if_incomplete keeps us off a half-paid state if the card declines.
      //
      // subscriptions.create only consults the customer's DEFAULT payment
      // method, and a card that arrived via subscription-mode Checkout is
      // attached but recorded as the (now possibly cancelled) subscription's
      // default, not the customer's — so a cancel-then-resubscribe would die
      // with "no attached payment source". Resolve one explicitly: use the
      // customer default when set, else the most recently attached method.
      let defaultPaymentMethod: string | undefined;
      const customer = await stripe.customers.retrieve(ctx.workspace.stripeCustomerId);
      const hasCustomerDefault =
        !customer.deleted &&
        Boolean(customer.invoice_settings?.default_payment_method || customer.default_source);
      if (!hasCustomerDefault) {
        const attached = await stripe.customers.listPaymentMethods(ctx.workspace.stripeCustomerId, {
          limit: 1,
        });
        defaultPaymentMethod = attached.data[0]?.id;
        if (!defaultPaymentMethod) {
          // BAD_REQUEST on purpose: the projects-landing subscriber treats it
          // as "card problem" and routes the user to Stripe checkout to add one.
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: "No payment method on file. Add one to subscribe.",
          });
        }
      }

      const createParams: Stripe.SubscriptionCreateParams = {
        customer: ctx.workspace.stripeCustomerId,
        items,
        ...(defaultPaymentMethod ? { default_payment_method: defaultPaymentMethod } : {}),
        billing_cycle_anchor_config: { day_of_month: 1 },
        // Pin classic billing mode (clover defaults new subscriptions to
        // "flexible", which itemizes prorations differently); see the same
        // pin in createSubscription.
        billing_mode: { type: "classic" },
        proration_behavior: "always_invoice",
        payment_behavior: "error_if_incomplete",
      };

      let sub: Stripe.Subscription;
      try {
        // Deterministic idempotency key: if Stripe created (and charged) the
        // subscription but the workspace write below failed, a retry replays
        // the SAME subscription instead of charging a second one. Unlike the
        // checkout-session path there is no webhook backstop for this create —
        // customer.subscription.created resolves the workspace by
        // stripeSubscriptionId, which was never written.
        //
        // Stripe's idempotency layer replays the CREATION-TIME response, so
        // after a full subscribe→cancel cycle inside the key window a replay
        // hands back the canceled subscription still claiming to be active.
        // Re-retrieve the live status and, on a corpse, chain a fresh key off
        // its id and mint a new subscription (a retry of THIS request replays
        // that same fresh subscription) — mirroring stripe/checkout/page.tsx.
        let idempotencyKey = `deploy-subscribe:${ctx.workspace.id}:${input.plan}`;
        sub = await stripe.subscriptions.create(createParams, { idempotencyKey });
        for (let attempt = 0; attempt < 3; attempt++) {
          const live = await stripe.subscriptions.retrieve(sub.id);
          if (!isDeadSubscription(live)) {
            sub = live;
            break;
          }
          idempotencyKey = `${idempotencyKey}:${live.id}`;
          sub = await stripe.subscriptions.create(createParams, { idempotencyKey });
        }
      } catch (err) {
        throw toBillingError(err);
      }

      if (sub.status !== "active" && sub.status !== "trialing") {
        try {
          await stripe.subscriptions.cancel(sub.id);
        } catch (cancelErr) {
          console.error(
            `Failed to cancel non-active subscription ${sub.id} after creation:`,
            cancelErr,
          );
        }
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `Subscription was created but is not active (ID: ${sub.id}). Please contact support.`,
        });
      }

      // Link the new subscription so the customer.subscription.* webhook can
      // resolve this workspace.
      workspaceUpdate = { stripeSubscriptionId: sub.id, deployPlan: input.plan };
    }

    // One transaction so the plan write and its audit log commit together; a
    // failure in either rolls back the other.
    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaces)
        .set(workspaceUpdate)
        .where(eq(schema.workspaces.id, ctx.workspace.id));
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: `Subscribed to Compute ${input.plan} plan.`,
        resources: [],
        context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
      });
    });
  });

/**
 * Surfaces Stripe payment errors as actionable TRPCErrors; rethrows anything
 * else so genuine bugs are not masked as a card problem.
 */
function toBillingError(err: unknown): TRPCError {
  if (err instanceof Stripe.errors.StripeCardError || err instanceof Stripe.errors.StripeError) {
    return new TRPCError({
      code: "BAD_REQUEST",
      message:
        err.message ||
        "Payment could not be completed. Please update your payment method and try again.",
    });
  }
  throw err;
}
