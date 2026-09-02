import { ActorType } from "@/gen/proto/ctrl/v1/actor_pb";
import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const authorizeDeployment = workspaceProcedure
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
      },
    });

    if (!deployment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Deployment not found or access denied",
      });
    }

    const ctrl = createCtrlClient(DeployService);

    try {
      // ctrl writes the deployment.authorize audit entry from this actor, so an
      // authorization is audited only when it actually happened.
      await ctrl.authorizeDeployment({
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

      console.error("Authorize deployment request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to authorize deployment",
      });
    }
  });
