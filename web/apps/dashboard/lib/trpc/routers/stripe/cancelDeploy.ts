import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig, findDeployItems } from "@/lib/stripe/deployBilling";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

/**
 * Cancels Unkey Deploy immediately: clears deploy_plan so the dashboard gate
 * blocks new deploys, and removes the Deploy items (plan-fee + metered) from
 * Stripe now, with no refund of the prepaid plan-fee. If Deploy was the only
 * thing on the subscription (the free-tier path created it), the whole
 * subscription is cancelled now instead, since Stripe requires at least one
 * item. The webhook reconciles deploy_plan to the same value.
 *
 * This does NOT stop running compute and does NOT bill usage correctly. Nothing
 * in ctrl/krane reacts to the cancel, so instances keep running and keep
 * emitting meter events after the items are gone, and those events are
 * uncollectable. The Deploy-only path invoices usage up to now (best effort);
 * the mixed path cannot, because subscriptionItems.del has no invoice_now.
 * Either way, usage between cancel and actual shutdown is lost.
 *
 * Correct cancel can only bill as the last step of a teardown sequence (stop
 * compute, wait for drain, flush final meter events, then bill and remove
 * items). Tracked in ENG-2922.
 */
export const cancelDeploy = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
    const config = await deployBillingConfig();
    if (!config) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Unkey Deploy billing is not configured.",
      });
    }

    if (!ctx.workspace.stripeSubscriptionId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace has no Unkey Deploy plan to cancel.",
      });
    }

    const stripe = getStripeClient();
    const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);

    const deployItems = findDeployItems(config, sub.items.data);
    if (deployItems.length === 0) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace has no Unkey Deploy plan to cancel.",
      });
    }

    if (deployItems.length === sub.items.data.length) {
      // Deploy-only subscription: can't delete the last item, so cancel the
      // whole subscription now. prorate:false = no plan-fee refund; invoice_now
      // bills metered usage up to now. Usage after this point is not billed,
      // because compute is not torn down here (ENG-2922).
      await stripe.subscriptions.cancel(sub.id, { prorate: false, invoice_now: true });
    } else {
      // Mixed subscription: drop just the Deploy items now, keep the API items.
      // One subscriptions.update marking each Deploy item deleted, so the
      // removal is atomic: a per-item del loop that dies mid-way would leave
      // orphaned metered items still billing while deploy_plan is never cleared.
      // proration_behavior:none = no plan-fee refund. There is no invoice_now on
      // this path, so metered usage since the last invoice on these meters is
      // NOT billed; the teardown-then-bill flow in ENG-2922 is the real fix.
      await stripe.subscriptions.update(sub.id, {
        items: deployItems.map((item) => ({ id: item.id, deleted: true })),
        proration_behavior: "none",
      });
    }

    // Immediate cutoff: clear the plan now so access stops right away. One
    // transaction so the clear and its audit log commit together; a failure in
    // either rolls back the other. The resulting webhook reconciles to null.
    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaces)
        .set({ deployPlan: null })
        .where(eq(schema.workspaces.id, ctx.workspace.id));
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: "Cancelled Unkey Deploy.",
        resources: [],
        context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
      });
    });

    await invalidateWorkspaceCache(ctx.tenant.id);
  });
