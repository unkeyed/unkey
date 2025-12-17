import { insertAuditLogs } from "@/lib/audit";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

// Import service definition that you want to connect to.
import { DeploymentService } from "@/gen/proto/ctrl/v1/deployment_pb";

import { db } from "@/lib/db";
import { env } from "@/lib/env";
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
    // Validate that ctrl service URL is configured
    const { CTRL_URL, CTRL_API_KEY } = env();
    if (!CTRL_URL || !CTRL_API_KEY) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "ctrl service is not configured",
      });
    }
    // Here we make the client itself, combining the service
    // definition with the transport.
    const ctrl = createClient(
      DeploymentService,
      createConnectTransport({
        baseUrl: CTRL_URL,
        interceptors: [
          (next) => (req) => {
            req.header.set("Authorization", `Bearer ${CTRL_API_KEY}`);
            return next(req);
          },
        ],
      }),
    );
    try {
      // Verify the target deployment exists and belongs to this workspace
      const targetDeployment = await db.query.deployments.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.targetDeploymentId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          id: true,
          status: true,
        },
        with: {
          project: {
            columns: {
              id: true,
              name: true,
              liveDeploymentId: true,
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
      if (!targetDeployment.project.liveDeploymentId) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: `Project ${targetDeployment.project.name} doesn't have a live deployment to roll back.`,
        });
      }

      await ctrl
        .rollback({
          sourceDeploymentId: targetDeployment.project.liveDeploymentId,
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
