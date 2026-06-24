import { ActorType } from "@/gen/proto/ctrl/v1/actor_pb";
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
      where: (table, { and, eq }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
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
        actor: {
          id: ctx.user.id,
          type: ActorType.USER,
          remoteIp: ctx.audit.location,
          userAgent: ctx.audit.userAgent ?? "",
        },
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

    return { success: true, projectId: input.projectId };
  });
