import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getDeploymentRps = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(
    z.object({
      deploymentId: z.string(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      const deployment = await db.query.deployments.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          projectId: true,
          environmentId: true,
        },
      });

      if (!deployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found",
        });
      }

      try {
        const result = await clickhouse.sentinel.rps.byDeployment({
          workspaceId: ctx.workspace.id,
          projectId: deployment.projectId,
          deploymentId: input.deploymentId,
          environmentId: deployment.environmentId,
        });

        if (result.err) {
          console.warn("Failed to fetch deployment RPS from ClickHouse", result.err);
          return { avg_rps: 0 };
        }

        return { avg_rps: result.val[0]?.avg_rps ?? 0 };
      } catch (chError) {
        console.warn("Failed to fetch deployment RPS from ClickHouse", chError);
        return { avg_rps: 0 };
      }
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployment RPS",
      });
    }
  });
