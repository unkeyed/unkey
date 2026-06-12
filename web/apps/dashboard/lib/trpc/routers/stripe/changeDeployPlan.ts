import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { deployBillingConfig, findPlanFeeItem } from "@/lib/stripe/deployPlans";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

/**
 * Switches the workspace's Unkey Deploy plan by repricing the plan-fee item on
 * its subscription. Metered items are shared across plans, so they are left
 * untouched. Writes workspaces.deploy_plan optimistically for instant UI;
 * the subscription.updated webhook reconciles it to the same value.
 */
export const changeDeployPlan = workspaceProcedure
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

    if (!ctx.workspace.stripeSubscriptionId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace has no Compute plan to change.",
      });
    }

    const stripe = getStripeClient();
    const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);

    // Find the current plan-fee item by matching its price against the
    // configured plan-fee ids, rather than trusting any client input.
    const planFeeItem = findPlanFeeItem(config, sub.items.data);
    if (!planFeeItem) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace has no Compute plan to change.",
      });
    }

    if (planFeeItem.plan === input.plan) {
      // Already on the requested plan; nothing to do.
      return;
    }

    const newPriceId = config.planFeePriceIds[input.plan];
    try {
      await stripe.subscriptionItems.update(planFeeItem.id, {
        price: newPriceId,
        proration_behavior: "create_prorations",
        payment_behavior: "error_if_incomplete",
      });
    } catch (err) {
      if (
        err instanceof Stripe.errors.StripeCardError ||
        err instanceof Stripe.errors.StripeError
      ) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message:
            err.message ||
            "Payment could not be completed. Please update your payment method and try again.",
        });
      }
      throw err;
    }

    // Optimistic: reflect the new plan immediately; the webhook reconciles.
    await db
      .update(schema.workspaces)
      .set({ deployPlan: input.plan })
      .where(eq(schema.workspaces.id, ctx.workspace.id));

    await insertAuditLogs(db, {
      workspaceId: ctx.workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "workspace.update",
      description: `Changed Compute plan to ${input.plan}.`,
      resources: [],
      context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
    });

    await invalidateWorkspaceCache(ctx.tenant.id);
  });
