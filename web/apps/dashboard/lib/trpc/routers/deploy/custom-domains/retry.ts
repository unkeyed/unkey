import { CustomDomainService } from "@/gen/proto/ctrl/v1/custom_domain_pb";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const retryVerification = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
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
      where: (table, { eq, and }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
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
      where: (table, { eq, and }) =>
        and(
          eq(table.domain, input.domain),
          eq(table.projectId, input.projectId),
          eq(table.workspaceId, ctx.workspace.id),
        ),
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
      const response = await ctrl.retryVerification({
        workspaceId: ctx.workspace.id,
        projectId: input.projectId,
        domain: input.domain,
      });

      return { status: response.status };
    } catch (error) {
      console.error("Retry verification failed:", error);

      if (error instanceof Error && error.message.includes("not found")) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Domain not found",
        });
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retry verification",
      });
    }
  });
