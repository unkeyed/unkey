import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import {
  deployBillingConfig,
  deploySubscriptionItems,
  findDeployItems,
} from "@/lib/stripe/deployPlans";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

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
    const config = deployBillingConfig();
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

    if (ctx.workspace.stripeSubscriptionId) {
      // Existing subscription (e.g. a paid API plan): append the Deploy items.
      // Items not listed here are left untouched, so API items are preserved.
      const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);

      if (findDeployItems(config, sub.items.data).length > 0) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "Workspace already has Compute items on its subscription.",
        });
      }

      // Attach edge cases, decided in ENG-2876: Deploy items only attach to a
      // subscription that will actually keep billing them. Anything else gets
      // a precondition error here instead of a raw Stripe error (or worse, a
      // silent attach to a subscription that ends at the period roll).
      if (sub.status !== "active" && sub.status !== "trialing") {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: `Your subscription is ${sub.status.replace(/_/g, " ")}. Settle any outstanding invoices before subscribing to Compute.`,
        });
      }
      if (sub.cancel_at_period_end) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message:
            "Your subscription is set to cancel at the end of this period. Resubscribe to your API plan first, or wait until it ends to start Compute fresh.",
        });
      }
      if (sub.currency !== "usd") {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message:
            "Compute prices are in USD and cannot be added to a non-USD subscription. Contact support@unkey.com.",
        });
      }

      try {
        await stripe.subscriptions.update(sub.id, {
          items,
          proration_behavior: "always_invoice",
          payment_behavior: "error_if_incomplete",
        });
      } catch (err) {
        throw toBillingError(err);
      }

      // Optimistic: reflect the new plan immediately; the webhook reconciles.
      await db
        .update(schema.workspaces)
        .set({ deployPlan: input.plan })
        .where(eq(schema.workspaces.id, ctx.workspace.id));
    } else {
      // Free tier: create a subscription whose initial items are the Deploy set.
      // error_if_incomplete keeps us off a half-paid state if the card declines.
      let sub: Stripe.Subscription;
      try {
        sub = await stripe.subscriptions.create({
          customer: ctx.workspace.stripeCustomerId,
          items,
          // Midnight-UTC anchor so billing periods are exact calendar months;
          // see the matching comment in createSubscription.
          billing_cycle_anchor_config: { day_of_month: 1, hour: 0, minute: 0, second: 0 },
          // Pin classic billing mode (Stripe API 2025-09-30 "clover" and later
          // default new subscriptions to "flexible", which itemizes prorations
          // differently); see the same pin in createSubscription.
          billing_mode: { type: "classic" },
          proration_behavior: "always_invoice",
          payment_behavior: "error_if_incomplete",
        });
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
      // resolve this workspace, and optimistically set the plan so the UI
      // reflects it immediately.
      await db
        .update(schema.workspaces)
        .set({ stripeSubscriptionId: sub.id, deployPlan: input.plan })
        .where(eq(schema.workspaces.id, ctx.workspace.id));
    }

    await insertAuditLogs(db, {
      workspaceId: ctx.workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "workspace.update",
      description: `Subscribed to Compute ${input.plan} plan.`,
      resources: [],
      context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
    });

    await invalidateWorkspaceCache(ctx.tenant.id);
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
