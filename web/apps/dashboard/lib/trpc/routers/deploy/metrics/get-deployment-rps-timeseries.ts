import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getDeploymentRpsTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
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
        const result = await clickhouse.sentinel.rps.timeseries({
          workspaceId: ctx.workspace.id,
          projectId: deployment.projectId,
          deploymentId: input.deploymentId,
          environmentId: deployment.environmentId,
        });

        if (result.err) {
          console.warn("Failed to fetch deployment RPS timeseries from ClickHouse", result.err);
          return [];
        }

        return result.val;
      } catch (chError) {
        console.warn("Failed to fetch deployment RPS timeseries from ClickHouse", chError);
        return [];
      }
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployment RPS timeseries",
      });
    }
  });
