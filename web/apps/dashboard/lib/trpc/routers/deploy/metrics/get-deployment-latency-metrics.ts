import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { percentileSchema } from "@unkey/clickhouse/src/sentinel";
import { z } from "zod";

const WINDOW_HOURS = 6;
const INTERVAL_MS = 15 * 60 * 1000;



export const getDeploymentLatencyMetrics = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      deploymentId: z.string(),
      percentile: percentileSchema,
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
        const result = await clickhouse.sentinel.latency.withTimeseries({
          workspaceId: ctx.workspace.id,
          projectId: deployment.projectId,
          deploymentId: input.deploymentId,
          environmentId: deployment.environmentId,
          percentile: input.percentile,
        });

        if (result.err) {
          console.warn("Failed to fetch deployment latency metrics from ClickHouse", result.err);
          return { current: 0, timeseries: [] };
        }

        // GROUP BY time WITH ROLLUP returns per-bucket rows plus one rollup
        // row where time='1970-01-01' (x=0) containing the exact global percentile.
        // This gives us current + timeseries in a single CH round trip.
        let current = 0;
        const raw: { x: number; y: number }[] = [];

        for (const row of result.val) {
          if (row.x === 0) {
            current = row.y;
          } else {
            raw.push(row);
          }
        }

        return { current, timeseries: fillGaps(raw) };
      } catch (chError) {
        console.warn("Failed to fetch deployment latency metrics from ClickHouse", chError);
        return { current: 0, timeseries: [] };
      }
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployment latency metrics",
      });
    }
  });




// The ROLLUP query skips WITH FILL (they conflict), so we fill empty
// 15-min slots in JS. Matches the CH WITH FILL range: [start, now) exclusive.
function fillGaps(points: { x: number; y: number }[]): { x: number; y: number }[] {
  const now = Date.now();
  const start = now - WINDOW_HOURS * 60 * 60 * 1000;
  const firstBucket = Math.floor(start / INTERVAL_MS) * INTERVAL_MS;
  const lastBucket = Math.floor(now / INTERVAL_MS) * INTERVAL_MS;

  const byTimestamp = new Map<number, number>();
  for (const p of points) {
    byTimestamp.set(p.x, p.y);
  }

  const filled: { x: number; y: number }[] = [];
  for (let t = firstBucket; t < lastBucket; t += INTERVAL_MS) {
    filled.push({ x: t, y: byTimestamp.get(t) ?? 0 });
  }
  return filled;
}
