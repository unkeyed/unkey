import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { insertAuditLogs } from "@/lib/audit";
import { deactivateNonCreatorMemberships } from "@/lib/auth/deactivateNonCreatorMemberships";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { cancelDeploySubscription } from "@/lib/stripe/cancelDeploySubscription";
import { deployPlanGrantsTeam } from "@/lib/stripe/deployPlan";
import { setComputeQuotas } from "@/lib/stripe/setComputeQuotas";
import { TRPCError } from "@trpc/server";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

/**
 * Cancels Unkey Deploy. Stops the Stripe renewal here in the dashboard (cancel
 * at period end for a Deploy-only subscription, remove the plan-fee item from a
 * mixed one, never refunding), then calls ctrl to tear down the workspace's
 * running compute and clear the deploy_plan entitlement. The Stripe logic lives
 * here alongside subscribe and change-plan so subscription knowledge is not
 * duplicated in Go. Compute-owned quotas return to their defaults, while any API
 * plan quotas remain intact. The audit log stays here so the user actor is recorded.
 */
export const cancelDeploy = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
    // Stop the Stripe renewal first. A workspace with a Deploy plan but no
    // subscription (a comped override) has no renewal to stop and goes straight
    // to teardown. Cancelling the Deploy subscription is now a native whole-
    // subscription cancel at period end, so no billing-config lookup is needed.
    const subscriptionId = ctx.workspace.stripeDeploySubscriptionId;
    if (subscriptionId) {
      try {
        await cancelDeploySubscription(getStripeClient(), subscriptionId);
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

    await db.transaction(async (tx) => {
      await setComputeQuotas(tx, {
        workspaceId: ctx.workspace.id,
        plan: null,
        preserveApiQuotas: ctx.workspace.tier !== "Free",
      });
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: "Cancelled Compute.",
        resources: [],
        context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
      });
    });

    if (ctx.workspace.tier === "Free" && deployPlanGrantsTeam(ctx.workspace.deployPlan)) {
      await deactivateNonCreatorMemberships(ctx.workspace.orgId);
    }
  });
