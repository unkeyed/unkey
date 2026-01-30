import { CustomDomainService } from "@/gen/proto/ctrl/v1/custom_domain_pb";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
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
    const { CTRL_URL, CTRL_API_KEY } = env();
    if (!CTRL_URL || !CTRL_API_KEY) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "ctrl service is not configured",
      });
    }

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

    const ctrl = createClient(
      CustomDomainService,
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
