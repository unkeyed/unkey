import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getDeploymentLatency = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(
    z.object({
      deploymentId: z.string(),
      percentile: z.string().default("p50"),
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
        const result = await clickhouse.sentinel.latency.byDeployment({
          workspaceId: ctx.workspace.id,
          projectId: deployment.projectId,
          deploymentId: input.deploymentId,
          environmentId: deployment.environmentId,
          percentile: input.percentile,
        });

        if (result.err) {
          console.warn("Failed to fetch deployment latency from ClickHouse", result.err);
          return { latency: 0 };
        }

        return { latency: result.val[0]?.latency ?? 0 };
      } catch (chError) {
        console.warn("Failed to fetch deployment latency from ClickHouse", chError);
        return { latency: 0 };
      }
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployment latency",
      });
    }
  });
