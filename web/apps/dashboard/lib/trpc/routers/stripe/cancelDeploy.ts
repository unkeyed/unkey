import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig, findDeployItems } from "@/lib/stripe/deployPlans";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

/**
 * Cancels Unkey Deploy immediately (like Railway / Fly), not at period end.
 * Deploy is usage-metered, so leaving it active until period end would let a
 * customer cancel, pull their card, and keep running up uncollectable usage.
 * Immediate cutoff caps that exposure.
 *
 * Removes the Deploy items (plan-fee + metered) now with no refund of the
 * prepaid plan-fee; metered usage consumed so far is still billed. If Deploy was
 * the only thing on the subscription (the free-tier path created it), the whole
 * subscription is cancelled now instead, since Stripe requires a subscription to
 * keep at least one item. Either way deploy_plan is cleared immediately so access
 * stops now; the webhook reconciles to the same value.
 */
export const cancelDeploy = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
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
        message: "Workspace has no Compute plan to cancel.",
      });
    }

    const stripe = getStripeClient();
    const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);

    const deployItems = findDeployItems(config, sub.items.data);
    if (deployItems.length === 0) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace has no Compute plan to cancel.",
      });
    }

    if (deployItems.length === sub.items.data.length) {
      // Deploy-only subscription: can't delete the last item, so cancel the
      // whole subscription now. prorate:false = no plan-fee refund; invoice_now
      // bills the metered usage consumed up to this point.
      await stripe.subscriptions.cancel(sub.id, { prorate: false, invoice_now: true });
    } else {
      // Mixed subscription: drop just the Deploy items now, keep the API items.
      // proration_behavior:none = no refund of the prepaid plan-fee.
      for (const item of deployItems) {
        await stripe.subscriptionItems.del(item.id, { proration_behavior: "none" });
      }
    }

    // Immediate cutoff: clear the plan now so access stops right away. The
    // resulting webhook reconciles to the same value.
    await db
      .update(schema.workspaces)
      .set({ deployPlan: null })
      .where(eq(schema.workspaces.id, ctx.workspace.id));

    await insertAuditLogs(db, {
      workspaceId: ctx.workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "workspace.update",
      description: "Cancelled Compute.",
      resources: [],
      context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
    });

    await invalidateWorkspaceCache(ctx.tenant.id);
  });
