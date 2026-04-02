import { insertAuditLogs } from "@/lib/audit";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { getCtrlClients } from "../../ctrl";

export const deleteProject = workspaceProcedure
  .input(
    z.object({
      projectId: z.string(),
    }),
  )
  .use(withRatelimit(ratelimit.delete))
  .mutation(async ({ ctx, input }) => {
    const project = await db.query.projects.findFirst({
      where: { id: input.projectId, workspaceId: ctx.workspace.id },
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found",
      });
    }

    if (project.deleteProtection) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Cannot delete project with delete protection enabled",
      });
    }

    const ctrl = getCtrlClients();

    try {
      await ctrl.project.deleteProject({
        projectId: input.projectId,
      });
    } catch (err) {
      if (err instanceof TRPCError) {
        throw err;
      }
      console.error("Failed to delete project:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to delete project",
      });
    }

    await insertAuditLogs(db, {
      workspaceId: ctx.workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "project.delete",
      description: `Deleted ${project.id}`,
      resources: [
        {
          type: "project",
          id: project.id,
          name: project.name,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return { success: true, projectId: input.projectId };
  });
