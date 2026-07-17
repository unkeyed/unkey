import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { insertAuditLogs } from "@/lib/audit";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { cancelDeploySubscription } from "@/lib/stripe/cancelDeploySubscription";
import { deployBillingConfig } from "@/lib/stripe/deployBilling";
import { TRPCError } from "@trpc/server";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

/**
 * Cancels Unkey Deploy. Stops the Stripe renewal here in the dashboard (cancel
 * at period end for a Deploy-only subscription, remove the plan-fee item from a
 * mixed one, never refunding), then calls ctrl to tear down the workspace's
 * running compute and clear the deploy_plan entitlement. The Stripe logic lives
 * here alongside subscribe and change-plan so subscription knowledge is not
 * duplicated in Go. The audit log stays here so the user actor is recorded.
 */
export const cancelDeploy = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
    // Stop the Stripe renewal first. A workspace with a Deploy plan but no
    // subscription (a comped override) has no renewal to stop and goes straight
    // to teardown.
    const subscriptionId = ctx.workspace.stripeSubscriptionId;
    if (subscriptionId) {
      // The renewal must be stopped before ctrl clears the entitlement. If the
      // billing config can't resolve (unconfigured, or a transient Stripe/reprice
      // window), fail the whole cancel rather than clearing deploy_plan while the
      // subscription keeps auto-renewing the plan fee.
      const config = await deployBillingConfig();
      if (!config) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Compute billing is unavailable right now. Please try again.",
        });
      }
      try {
        await cancelDeploySubscription(getStripeClient(), subscriptionId, config);
      } catch (error) {
        console.error("Stripe cancel for Compute failed:", error);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to cancel Compute billing.",
        });
      }
    }

    // ctrl tears down running compute and clears the local entitlement. It runs
    // after the Stripe cancel and is safe to retry: teardown is keyed and
    // idempotent and the entitlement clear is a pure set, so a failure here
    // leaves billing already stopped and a retry finishes the teardown.
    const ctrl = createCtrlClient(DeployService);
    try {
      await ctrl.deprovisionCompute({ workspaceId: ctx.workspace.id });
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      console.error("Cancel Compute request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to cancel Compute.",
      });
    }

    await insertAuditLogs(db, {
      workspaceId: ctx.workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "workspace.update",
      description: "Cancelled Compute.",
      resources: [],
      context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
    });
  });
