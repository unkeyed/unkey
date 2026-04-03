import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { insertAuditLogs } from "@/lib/audit";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";

import { z } from "zod";

export const rollback = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      targetDeploymentId: z.string().min(1, "Target deployment ID is required"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ctrl = createCtrlClient(DeployService);

    try {
      // Verify the target deployment exists and belongs to this workspace
      const targetDeployment = await db.query.deployments.findFirst({
        where: { id: input.targetDeploymentId, workspaceId: ctx.workspace.id },
        columns: {
          id: true,
          status: true,
          appId: true,
          environmentId: true,
        },
        with: {
          project: {
            columns: {
              id: true,
              name: true,
            },
          },
        },
      });

      if (!targetDeployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found or access denied",
        });
      }

      if (targetDeployment.status !== "ready") {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: `Deployment ${targetDeployment.id} is not ready (status: ${targetDeployment.status})`,
        });
      }

      // Get currentDeploymentId from the app
      const app = await db.query.apps.findFirst({
        where: { id: targetDeployment.appId, workspaceId: ctx.workspace.id },
        columns: { currentDeploymentId: true },
      });

      if (!app?.currentDeploymentId) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: `Project ${targetDeployment.project.name} doesn't have a current deployment to roll back.`,
        });
      }

      await ctrl
        .rollback({
          sourceDeploymentId: app.currentDeploymentId,
          targetDeploymentId: targetDeployment.id,
        })
        .catch((err) => {
          console.error(err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: err,
          });
        });

      // Log the rollback action for audit purposes
      await insertAuditLogs(db, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "deployment.rollback",
        description: `Rolled back ${targetDeployment.project.name} to deployment ${targetDeployment.id}`,
        resources: [
          {
            type: "deployment",
            id: input.targetDeploymentId,
            name: targetDeployment.project.name,
          },
        ],
        context: ctx.audit,
      });

      return {};
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      console.error("Rollback request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to communicate with control service",
      });
    }
  });
