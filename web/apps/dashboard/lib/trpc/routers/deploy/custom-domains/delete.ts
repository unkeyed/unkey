import { CustomDomainService } from "@/gen/proto/ctrl/v1/custom_domain_pb";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const deleteCustomDomain = workspaceProcedure
  .use(withRatelimit(ratelimit.delete))
  .input(
    z.object({
      domain: z.string().min(1, "Domain is required"),
      projectId: z.string().min(1, "Project ID is required"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ctrl = createCtrlClient(CustomDomainService);

    // Verify project belongs to workspace
    const project = await db.query.projects.findFirst({
      where: { id: input.projectId, workspaceId: ctx.workspace.id },
      columns: {
        id: true,
      },
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found",
      });
    }

    // Verify domain belongs to project and workspace
    const customDomain = await db.query.customDomains.findFirst({
      where: { domain: input.domain, projectId: input.projectId, workspaceId: ctx.workspace.id },
      columns: {
        id: true,
      },
    });

    if (!customDomain) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Domain not found for project",
      });
    }

    try {
      await ctrl.deleteCustomDomain({
        workspaceId: ctx.workspace.id,
        projectId: input.projectId,
        domain: input.domain,
      });

      return { success: true };
    } catch (error) {
      console.error("Delete custom domain failed:", error);

      if (error instanceof Error && error.message.includes("not found")) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Domain not found",
        });
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to delete custom domain",
      });
    }
  });
