import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

// Returns the rolling average RPS across every instance of a deployment in
// the given region. The region card shows this single number; per-instance
// breakdown stays on instance nodes via getInstanceRps.
export const getRegionRps = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      deploymentId: z.string(),
      region: z.string(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      const deployment = await db.query.deployments.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          environmentId: true,
          projectId: true,
        },
      });

      if (!deployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found",
        });
      }

      try {
        const result = await clickhouse.sentinel.rps.byRegion({
          workspaceId: ctx.workspace.id,
          deploymentId: input.deploymentId,
          environmentId: deployment.environmentId,
          projectId: deployment.projectId,
          region: input.region,
        });

        if (result.err) {
          console.warn("Failed to fetch region RPS from ClickHouse", result.err);
          return undefined;
        }

        return result.val[0]?.avg_rps;
      } catch (chError) {
        console.warn("Failed to fetch region RPS from ClickHouse", chError);
        return undefined;
      }
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch region RPS",
      });
    }
  });
