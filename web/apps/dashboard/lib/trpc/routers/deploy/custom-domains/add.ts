import { CustomDomainService } from "@/gen/proto/ctrl/v1/custom_domain_pb";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { Code, ConnectError } from "@connectrpc/connect";
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

    // Verify environment belongs to project and resolve the app
    const environment = await db.query.environments.findFirst({
      where: { id: input.environmentId, projectId: input.projectId },
      columns: {
        id: true,
        appId: true,
      },
    });

    if (!environment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Environment not found",
      });
    }

    const appId = environment.appId;

    try {
      const response = await ctrl.addCustomDomain({
        workspaceId: ctx.workspace.id,
        projectId: input.projectId,
        appId,
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

      if (error instanceof ConnectError && error.code === Code.AlreadyExists) {
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
