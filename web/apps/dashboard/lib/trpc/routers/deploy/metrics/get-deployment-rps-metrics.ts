import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getDeploymentRpsMetrics = workspaceProcedure
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

      const result = await clickhouse.sentinel.rps.timeseries({
        workspaceId: ctx.workspace.id,
        projectId: deployment.projectId,
        deploymentId: input.deploymentId,
        environmentId: deployment.environmentId,
      });

      if (result.err) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to fetch deployment RPS metrics",
          cause: result.err,
        });
      }

      // Average ALL buckets (including zeros) so RPS decays naturally
      // when traffic stops, matching total_requests / window_duration.
      const points = result.val;
      const current =
        points.length > 0
          ? Math.round((points.reduce((s, p) => s + p.y, 0) / points.length) * 100) / 100
          : 0;

      return { current, timeseries: points };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployment RPS metrics",
      });
    }
  });
