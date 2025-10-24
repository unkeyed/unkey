import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { requireUser, requireWorkspace, t } from "@/lib/trpc/trpc";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { TRPCError } from "@trpc/server";
import { type GetOpenApiDiffResponse, OpenApiService } from "@unkey/proto";
import { z } from "zod";

export const getOpenApiDiff = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      oldDeploymentId: z.string(),
      newDeploymentId: z.string(),
    }),
  )
  .query(async ({ input, ctx }) => {
    const { CTRL_URL, CTRL_API_KEY } = env();
    if (!CTRL_URL || !CTRL_API_KEY) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "ctrl service is not configured",
      });
    }

    // Here we make the client itself, combining the service
    // definition with the transport.
    const ctrl = createClient(
      OpenApiService,
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
      const deployments = await db.query.deployments.findMany({
        where: (table, { and, eq, inArray }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            inArray(table.id, [input.oldDeploymentId, input.newDeploymentId]),
          ),
      });
      if (deployments.length !== 2) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "One or both deployments not found",
        });
      }
      if (deployments[0].projectId !== deployments[1].projectId) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Deployments must belong to the same project",
        });
      }

      const resp = await ctrl.getOpenApiDiff({
        oldDeploymentId: input.oldDeploymentId,
        newDeploymentId: input.newDeploymentId,
      });

      return {
        hasBreakingChanges: (resp as GetOpenApiDiffResponse).hasBreakingChanges,
        summary: (resp as GetOpenApiDiffResponse).summary,
        changes: (resp as GetOpenApiDiffResponse).changes,
      };
    } catch (error) {
      console.error("Failed to get OpenAPI diff:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to get OpenAPI diff",
      });
    }
  });
