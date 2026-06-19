import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { TIMESERIES_INTERVAL_MINUTES } from "@unkey/clickhouse/src/sentinel";
import { z } from "zod";

const INTERVAL_MS = TIMESERIES_INTERVAL_MINUTES * 60 * 1000;
const HOUR_MS = 60 * 60 * 1000;
const DAY_MS = 24 * HOUR_MS;

type Range = "hour" | "day" | "week";
type Point = { time: number; requests: number; errors: number };

function selectRange(ageMs: number): { range: Range; windowHours: number } {
  if (ageMs < HOUR_MS) {
    return { range: "hour", windowHours: 1 };
  }
  if (ageMs < DAY_MS) {
    return { range: "day", windowHours: 24 };
  }
  return { range: "week", windowHours: 24 * 7 };
}

export const getAppRpsMetrics = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ appId: z.string() }))
  .query(async ({ ctx, input }) => {
    const app = await db.query.apps.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.id, input.appId), eq(table.workspaceId, ctx.workspace.id)),
      columns: { projectId: true, createdAt: true },
    });

    if (!app) {
      throw new TRPCError({ code: "NOT_FOUND", message: "App not found" });
    }

    const { range, windowHours } = selectRange(Date.now() - app.createdAt);

    const appDeployments = await db.query.deployments.findMany({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, ctx.workspace.id), eq(table.appId, input.appId)),
      columns: { id: true },
    });
    const deploymentIds = appDeployments.map((d) => d.id);

    const empty = { range, totalRequests: 0, rps: fillGaps(new Map(), windowHours) };
    if (deploymentIds.length === 0) {
      return empty;
    }

    try {
      const result = await clickhouse.sentinel.requests.appTimeseries({
        workspaceId: ctx.workspace.id,
        projectId: app.projectId,
        deploymentIds,
        windowHours,
      });

      if (result.err) {
        console.warn("Failed to fetch app RPS metrics from ClickHouse", result.err);
        return empty;
      }

      const byBucket = new Map<number, { requests: number; errors: number }>();
      let totalRequests = 0;
      for (const row of result.val) {
        byBucket.set(row.x, { requests: row.requests, errors: row.errors });
        totalRequests += row.requests;
      }

      return { range, totalRequests, rps: fillGaps(byBucket, windowHours) };
    } catch (chError) {
      console.warn("Failed to fetch app RPS metrics from ClickHouse", chError);
      return empty;
    }
  });

function fillGaps(
  byBucket: Map<number, { requests: number; errors: number }>,
  windowHours: number,
): Point[] {
  const now = Date.now();
  const start = now - windowHours * HOUR_MS;
  const firstBucket = Math.floor(start / INTERVAL_MS) * INTERVAL_MS;
  const lastBucket = Math.floor(now / INTERVAL_MS) * INTERVAL_MS;

  const filled: Point[] = [];
  for (let t = firstBucket; t < lastBucket; t += INTERVAL_MS) {
    const v = byBucket.get(t);
    filled.push({ time: t, requests: v?.requests ?? 0, errors: v?.errors ?? 0 });
  }
  return filled;
}
