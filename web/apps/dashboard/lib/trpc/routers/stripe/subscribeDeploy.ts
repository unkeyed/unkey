import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import {
  deployBillingConfig,
  deploySubscriptionItems,
  findDeployItems,
} from "@/lib/stripe/deployBilling";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
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
      let sub: Stripe.Subscription;
      try {
        sub = await stripe.subscriptions.create({
          customer: ctx.workspace.stripeCustomerId,
          items,
          billing_cycle_anchor_config: { day_of_month: 1 },
          // Pin classic billing mode (clover defaults new subscriptions to
          // "flexible", which itemizes prorations differently); see the same
          // pin in createSubscription.
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
