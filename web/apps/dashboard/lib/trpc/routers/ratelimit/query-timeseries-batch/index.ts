import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { TIMESERIES_GRANULARITIES, getTimeseriesGranularity } from "../../utils/granularity";

const batchTimeseriesPayload = z.object({
  namespaceIds: z.array(z.string()).min(1).max(100),
  startTime: z.int(),
  endTime: z.int(),
});

type TimeseriesPoint = { x: number; y: { passed: number; total: number } };

/**
 * Generate a complete set of time buckets for the given range and step,
 * filling missing buckets with zeroes. This replicates the WITH FILL behavior
 * from the single-namespace query but works correctly per-namespace since
 * ClickHouse WITH FILL doesn't fill per-group in multi-dimensional GROUP BY.
 */
function fillGaps(
  points: TimeseriesPoint[],
  startTime: number,
  endTime: number,
  stepMs: number,
): TimeseriesPoint[] {
  // Align start/end to step boundaries (floor)
  const alignedStart = Math.floor(startTime / stepMs) * stepMs;
  const alignedEnd = Math.floor(endTime / stepMs) * stepMs + stepMs;

  // Index existing points by timestamp for O(1) lookup
  const pointMap = new Map<number, TimeseriesPoint>();
  for (const p of points) {
    pointMap.set(p.x, p);
  }

  const filled: TimeseriesPoint[] = [];
  for (let t = alignedStart; t < alignedEnd; t += stepMs) {
    filled.push(pointMap.get(t) ?? { x: t, y: { passed: 0, total: 0 } });
  }
  return filled;
}

export const queryRatelimitTimeseriesBatch = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(batchTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    const namespaces = await db.query.ratelimitNamespaces
      .findMany({
        where: (table, { and, eq, inArray, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            inArray(table.id, input.namespaceIds),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve ratelimit timeseries analytics. If this issue persists, please contact support@unkey.com.",
        });
      });

    const validNamespaceIds = namespaces.map((ns) => ns.id);
    if (validNamespaceIds.length === 0) {
      return { timeseriesByNamespace: {}, granularity: "perMinute" as const };
    }

    const { granularity, startTime, endTime } = getTimeseriesGranularity(
      "forRegular",
      input.startTime,
      input.endTime,
    );

    const result = await clickhouse.ratelimits.batchTimeseries[granularity]({
      workspaceId: ctx.workspace.id,
      namespaceIds: validNamespaceIds,
      startTime,
      endTime,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve ratelimit timeseries analytics. If this issue persists, please contact support@unkey.com.",
      });
    }

    // Group raw results by namespace_id
    const rawByNamespace: Record<string, TimeseriesPoint[]> = {};
    for (const nsId of validNamespaceIds) {
      rawByNamespace[nsId] = [];
    }
    for (const point of result.val) {
      if (rawByNamespace[point.namespace_id]) {
        rawByNamespace[point.namespace_id].push({ x: point.x, y: point.y });
      }
    }

    // Fill gaps per namespace so each has a complete set of time buckets
    const stepMs = TIMESERIES_GRANULARITIES[granularity].ms;
    const timeseriesByNamespace: Record<string, TimeseriesPoint[]> = {};
    for (const nsId of validNamespaceIds) {
      timeseriesByNamespace[nsId] = fillGaps(
        rawByNamespace[nsId],
        startTime,
        endTime,
        stepMs,
      );
    }

    return { timeseriesByNamespace, granularity };
  });
