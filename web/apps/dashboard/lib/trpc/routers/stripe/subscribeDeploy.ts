import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig, findPlanFeeItem } from "@/lib/stripe/deployBilling";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { isDeadSubscription } from "@/lib/stripe/subscriptionUtils";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

/**
 * Resumes a same-tier Compute subscription which is cancelling at period end.
 * New subscriptions are created in Checkout so first-payment CVC and 3DS can be
 * completed on-session; this mutation never creates or charges a subscription.
 *
 * Writes workspaces.deploy_plan optimistically so the UI reflects the new plan
 * immediately. Stripe stays source of truth: customer.subscription.* reconciles
 * deploy_plan on webhook retry.
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

    // Claim the Compute plan slot before touching Stripe. Two admins racing with
    // different plans both pass the deployPlan guard above (it reads a possibly
    // stale request snapshot) and Stripe only rejects duplicate SAME-price items,
    // so without this both plan-fee items would attach and both bill. The
    // compare-and-set on plan IS NULL serializes them: exactly one write matches a
    // row, the loser gets zero and bails before any Stripe charge. This also
    // writes the plan optimistically (the transaction below re-commits the same
    // value), so the UI reflects it immediately.
    const claim = await db
      .update(schema.workspaceBilling)
      .set({ plan: input.plan })
      .where(
        and(
          eq(schema.workspaceBilling.workspaceId, ctx.workspace.id),
          isNull(schema.workspaceBilling.plan),
        ),
      );
    if (claim[0].affectedRows !== 1) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace already has a Compute plan. Use change instead.",
      });
    }

    // Release the claim if anything below fails, so a genuine retry can proceed.
    // Conditioned on the plan we set, so a webhook that reconciled a different
    // value in the meantime is never clobbered.
    const releaseClaim = async () => {
      await db
        .update(schema.workspaceBilling)
        .set({ plan: null })
        .where(
          and(
            eq(schema.workspaceBilling.workspaceId, ctx.workspace.id),
            eq(schema.workspaceBilling.plan, input.plan),
          ),
        );
    };

    try {
      if (!ctx.workspace.stripeDeploySubscriptionId) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "Start a new Compute subscription through Checkout.",
        });
      }

      const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeDeploySubscriptionId);
      if (isDeadSubscription(sub)) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "Start a new Compute subscription through Checkout.",
        });
      }
      if (!sub.cancel_at_period_end) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "Workspace already has a Compute plan. Use change instead.",
        });
      }

      // Resume only at the same tier. Clearing cancellation creates no invoice,
      // so there is no payment or customer action to recover here.
      const planFeeItem = findPlanFeeItem(config, sub.items.data);
      if (!planFeeItem || planFeeItem.plan !== input.plan) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: planFeeItem
            ? `Your cancelled Compute plan is ${planFeeItem.plan}. Resume it by subscribing to ${planFeeItem.plan} again, then switch to ${input.plan}.`
            : "Your cancelled Compute subscription has no plan fee to resume. Contact support@unkey.com.",
        });
      }

      try {
        await stripe.subscriptions.update(sub.id, { cancel_at_period_end: false });
      } catch (err) {
        throw toBillingError(err);
      }

      // One transaction so the plan write and its audit log commit together; a
      // failure in either rolls back the other.
      await db.transaction(async (tx) => {
        await tx
          .update(schema.workspaceBilling)
          .set({ plan: input.plan })
          .where(eq(schema.workspaceBilling.workspaceId, ctx.workspace.id));
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.update",
          description: `Subscribed to Compute ${input.plan} plan.`,
          resources: [],
          context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
        });
      });
    } catch (err) {
      await releaseClaim().catch((releaseErr) => {
        console.error("Failed to release Compute plan claim after subscribe error", {
          workspaceId: ctx.workspace.id,
          error: releaseErr instanceof Error ? releaseErr.message : releaseErr,
        });
      });
      throw err;
    }
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
