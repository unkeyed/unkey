import { CustomDomainService } from "@/gen/proto/ctrl/v1/custom_domain_pb";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const addCustomDomain = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(
    z.object({
      projectId: z.string().min(1, "Project ID is required"),
      environmentId: z.string().min(1, "Environment ID is required"),
      domain: z.string().min(1, "Domain is required").max(253, "Domain too long"),
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

    // Verify environment belongs to project
    const environment = await db.query.environments.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.id, input.environmentId), eq(table.projectId, input.projectId)),
      columns: {
        id: true,
      },
    });

    if (!environment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Environment not found",
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
      const response = await ctrl.addCustomDomain({
        workspaceId: ctx.workspace.id,
        projectId: input.projectId,
        environmentId: input.environmentId,
        domain: input.domain,
      });

      return {
        domainId: response.domainId,
        targetCname: response.targetCname,
        status: response.status,
      };
    } catch (error) {
      console.error("Add custom domain failed:", error);

      // Check if it's an already exists error
      if (error instanceof Error && error.message.includes("already")) {
        throw new TRPCError({
          code: "CONFLICT",
          message: "Domain already registered",
        });
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to add custom domain",
      });
    }
  });
