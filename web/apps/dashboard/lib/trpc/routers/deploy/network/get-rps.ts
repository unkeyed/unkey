import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

// The network view draws a card for every region and every instance of a
// deployment. Both breakdowns are served from one aggregation so a refresh
// costs a single deployment lookup and a single ClickHouse query, however
// many cards the canvas renders.
export const getRps = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ deploymentId: z.string() }))
  .output(
    z.object({
      regions: z.record(z.string(), z.number()),
      instances: z.record(z.string(), z.number()),
    }),
  )
  .query(async ({ ctx, input }) => {
    const empty: { regions: Record<string, number>; instances: Record<string, number> } = {
      regions: {},
      instances: {},
    };

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
      const result = await clickhouse.frontline.rps.breakdown({
        workspaceId: ctx.workspace.id,
        deploymentId: input.deploymentId,
        environmentId: deployment.environmentId,
        projectId: deployment.projectId,
      });

      if (result.err) {
        console.warn("Failed to fetch deployment RPS from ClickHouse", result.err);
        return empty;
      }

      const instances: Record<string, number> = {};
      const regions: Record<string, number> = {};

      for (const row of result.val) {
        instances[row.instance_id] = row.avg_rps;
        regions[row.region] = Math.round(((regions[row.region] ?? 0) + row.avg_rps) * 100) / 100;
      }

      return { regions, instances };
    } catch (chError) {
      console.warn("Failed to fetch deployment RPS from ClickHouse", chError);
      return empty;
    }
  });
