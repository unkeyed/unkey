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
      projectId: z.string().min(1, "Project ID is required"),
      branch: z.string().min(1, "Branch is required"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ctrl = createCtrlClient(DeployService);

    const project = await db.query.projects.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      columns: {
        id: true,
        name: true,
      },
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found or access denied",
      });
    }

    try {
      await ctrl.authorizeDeployment({
        projectId: input.projectId,
        branch: input.branch,
      });

      await insertAuditLogs(db, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "deployment.create",
        description: `Authorized deployment for ${project.name} on branch ${input.branch}`,
        resources: [
          {
            type: "project",
            id: input.projectId,
            name: project.name,
          },
        ],
        context: ctx.audit,
      });

      return {};
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      const message = error instanceof Error ? error.message : String(error);
      console.error("Authorize deployment request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: `Failed to authorize deployment: ${message}`,
      });
    }
  });
