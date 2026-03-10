import { createCtrlClient } from "@/lib/ctrl-client";
import { OpenApiService } from "@/gen/proto/ctrl/v1/openapi_pb";
import { db } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getOpenApiDiff = workspaceProcedure
  .input(
    z.object({
      oldDeploymentId: z.string(),
      newDeploymentId: z.string(),
    }),
  )
  .query(async ({ input, ctx }) => {
    const ctrl = createCtrlClient(OpenApiService);

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
        hasBreakingChanges: resp.hasBreakingChanges,
        summary: resp.summary,
        changes: resp.changes,
      };
    } catch (error) {
      console.error("Failed to get OpenAPI diff:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to get OpenAPI diff",
      });
    }
  });
