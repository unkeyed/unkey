import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { insertAuditLogs } from "@/lib/audit";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

/**
 * Cancels Unkey Deploy through ctrl-api. The RPC stops the Stripe renewal
 * (cancel at period end for a Deploy-only subscription, or removes the plan-fee
 * item from a mixed subscription, never refunding), clears the deploy_plan
 * entitlement, and tears down the workspace's running compute. The audit log
 * stays here so the user actor is recorded.
 */
export const cancelDeploy = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
    const ctrl = createCtrlClient(DeployService);

    try {
      await ctrl.cancelDeploy({ workspaceId: ctx.workspace.id });
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
