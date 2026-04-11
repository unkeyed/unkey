import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { insertAuditLogs } from "@/lib/audit";
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
        projectId: true,
      },
      with: {
        project: {
          columns: {
            name: true,
          },
        },
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
      await ctrl.cancelDeployment({
        deploymentId: input.deploymentId,
      });

      await insertAuditLogs(db, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "deployment.cancel",
        description: `Cancelled deployment ${input.deploymentId} for ${deployment.project.name}`,
        resources: [
          {
            type: "deployment",
            id: deployment.id,
            name: deployment.project.name,
          },
        ],
        context: ctx.audit,
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
