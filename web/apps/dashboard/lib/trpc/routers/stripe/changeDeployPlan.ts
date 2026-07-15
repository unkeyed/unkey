import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig, findDeployItems, findPlanFeeItem } from "@/lib/stripe/deployBilling";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import {
  createApiCancelSchedule,
  getApiCancelSchedule,
  isPendingSchedule,
} from "@/lib/stripe/subscriptionUtils";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

/**
 * Switches the workspace's Unkey Deploy plan by repricing the plan-fee item on
 * its subscription. Metered items are shared across plans, so they are left
 * untouched. Writes workspaces.deploy_plan optimistically for instant UI;
 * the subscription.updated webhook reconciles it to the same value.
 *
 * Upgrades charge the prorated fee difference immediately (always_invoice).
 * Paying that invoice triggers the credit-grant webhook, which tops up the
 * period's usage credits by the same net amount, so an upgrade buys more
 * credits for the rest of the month, not just a bigger fee.
 *
 * Downgrades keep the period as bought: no proration at all, so there is no
 * refund of the fee difference AND no clawback of the usage credits that fee
 * already granted. The customer rides out the period at the level they paid
 * for; the lower fee (and its smaller credits) starts at the next renewal.
 */
export const changeDeployPlan = workspaceProcedure
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

    // A pending API-plan cancellation is a schedule whose next phase snapshots
    // the CURRENT Compute items — left in place, it would revert the plan
    // change at the period boundary. Release it, apply the change, then
    // recreate it from the updated items so the pending cancellation survives.
    const apiCancelSchedule = await getApiCancelSchedule(stripe, sub);
    const pendingApiCancel = apiCancelSchedule !== null && isPendingSchedule(apiCancelSchedule);
    if (pendingApiCancel && apiCancelSchedule) {
      await stripe.subscriptionSchedules.release(apiCancelSchedule.id);
    }

    const restoreApiCancel = async () => {
      const updated = await stripe.subscriptions.retrieve(sub.id);
      const deployItemIds = new Set(
        findDeployItems(config, updated.items.data).map((item) => item.id),
      );
      await createApiCancelSchedule(
        stripe,
        sub.id,
        updated.items.data.filter((item) => deployItemIds.has(item.id)),
      );
    };

    const newPriceId = config.planFeePriceIds[input.plan];
    try {
      // DEPLOY_PLANS is ordered lowest to highest, so plan order doubles as
      // the upgrade/downgrade direction.
      const isDowngrade = DEPLOY_PLANS.indexOf(input.plan) < DEPLOY_PLANS.indexOf(planFeeItem.plan);

      await stripe.subscriptionItems.update(planFeeItem.id, {
        price: newPriceId,
        proration_behavior: isDowngrade ? "none" : "always_invoice",
        payment_behavior: "error_if_incomplete",
      });
    } catch (err) {
      if (pendingApiCancel) {
        // The item is unchanged (the update failed), so put the pending
        // cancellation back. Best-effort: losing it means the API plan keeps
        // billing until the user cancels again, not a wrong charge.
        try {
          await restoreApiCancel();
        } catch (restoreErr) {
          console.error("Failed to restore pending API cancellation after plan-change error", {
            subscriptionId: sub.id,
            error: restoreErr instanceof Error ? restoreErr.message : restoreErr,
          });
        }
      }
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

    if (pendingApiCancel) {
      // The plan change already went through, so a throw here would skip the
      // DB write below and a same-plan retry would short-circuit before ever
      // recreating the schedule. Best-effort instead: losing the pending
      // cancellation means the API plan keeps billing until the user cancels
      // again, not a wrong charge.
      try {
        await restoreApiCancel();
      } catch (restoreErr) {
        console.error("Failed to restore pending API cancellation after plan change", {
          subscriptionId: sub.id,
          error: restoreErr instanceof Error ? restoreErr.message : restoreErr,
        });
      }
    }

    // One transaction so the plan write and its audit log commit together; a
    // failure in either rolls back the other. Written optimistically; the
    // subscription.updated webhook reconciles deploy_plan to the same value.
    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaces)
        .set({ deployPlan: input.plan })
        .where(eq(schema.workspaces.id, ctx.workspace.id));
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: `Changed Compute plan to ${input.plan}.`,
        resources: [],
        context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
      });
    });
  });
