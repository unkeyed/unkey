import { insertAuditLogs } from "@/lib/audit";
import { createCtrlClient } from "@/lib/ctrl-client";

import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";

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

    const ctrl = createCtrlClient(DeployService);

    try {
      await ctrl.authorizeDeployment({
        deploymentId: input.deploymentId,
      });

      await insertAuditLogs(db, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "deployment.authorize",
        description: `Authorized deployment ${input.deploymentId} for ${deployment.project.name}`,
        resources: [
          {
            type: "project",
            id: deployment.projectId,
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

      console.error("Authorize deployment request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to authorize deployment",
      });
    }
  });
