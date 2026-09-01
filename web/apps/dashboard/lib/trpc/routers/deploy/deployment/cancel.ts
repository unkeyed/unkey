import { ActorType } from "@/gen/proto/ctrl/v1/actor_pb";
import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

// Deployments in these statuses are terminal — cancelling them is a no-op
// at the backend, so we reject from the UI upfront to avoid noise in the
// audit log.
const TERMINAL_STATUSES = new Set(["ready", "failed", "skipped", "stopped"]);

export const cancelDeployment = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      deploymentId: z.string().min(1, "Deployment ID is required"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const deployment = await db.query.deployments.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
      columns: {
        id: true,
        status: true,
      },
    });

    if (!deployment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Deployment not found or access denied",
      });
    }

    if (TERMINAL_STATUSES.has(deployment.status)) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `Deployment is already ${deployment.status}`,
      });
    }

    const ctrl = createCtrlClient(DeployService);

    try {
      // ctrl writes the deployment.cancel audit entry from this actor, so a
      // cancel is audited if and only if it actually happened.
      await ctrl.cancelDeployment({
        deploymentId: input.deploymentId,
        actor: {
          id: ctx.user.id,
          type: ActorType.USER,
          remoteIp: ctx.audit.location,
          userAgent: ctx.audit.userAgent ?? "",
        },
      });

      return {};
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      console.error("Cancel deployment request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to cancel deployment",
      });
    }
  });
