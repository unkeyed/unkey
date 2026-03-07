import { insertAuditLogs } from "@/lib/audit";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";

import { db } from "@/lib/db";
import { env } from "@/lib/env";
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
    const { CTRL_URL, CTRL_API_KEY } = env();
    if (!CTRL_URL || !CTRL_API_KEY) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "ctrl service is not configured",
      });
    }

    // Verify the project belongs to this workspace
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

    const ctrl = createClient(
      DeployService,
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

      console.error("Authorize deployment request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to communicate with control service",
      });
    }
  });
