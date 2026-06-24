import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { selectEnvironmentRequestsQuery } from "./environment-requests-query";

// Point is one sample of the request-rate chart: `time` is the bucket start as
// Unix ms, `requests` and `errors` are per-second rates for that bucket.
type Point = { time: number; requests: number; errors: number };

// getAppRpsMetrics returns the request-rate series for an app overview chart,
// computed from its production environment's Frontline traffic. The result is
// { range, totalRequests, rps }: a coarse range label, the total request count
// over the window, and a dense per-second series.
//
// It throws NOT_FOUND when the app does not exist. It returns an empty series
// (totalRequests 0, rps []) when the app has no production environment or the
// ClickHouse read fails, so the overview degrades to an empty chart instead of
// erroring.
export const getAppRpsMetrics = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ appId: z.string() }))
  .query(async ({ ctx, input }) => {
    const app = await db.query.apps.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.id, input.appId), eq(table.workspaceId, ctx.workspace.id)),
      columns: { projectId: true, createdAt: true },
      with: {
        environments: {
          where: (table, { eq }) => eq(table.slug, "production"),
          columns: { id: true },
        },
      },
    });

    if (!app) {
      throw new TRPCError({ code: "NOT_FOUND", message: "App not found" });
    }

    const query = selectEnvironmentRequestsQuery(app.createdAt);
    const empty = { range: query.range, totalRequests: 0, rps: [] as Point[] };

    const productionEnvironment = app.environments.at(0);
    if (!productionEnvironment) {
      return empty;
    }

    const result = await clickhouse.environment.requests({
      workspaceId: ctx.workspace.id,
      projectId: app.projectId,
      appId: input.appId,
      environmentId: productionEnvironment.id,
      interval: query.interval,
      startTimeMs: query.startTimeMs,
      endTimeMs: query.endTimeMs,
    });

    if (result.err) {
      console.error("Failed to fetch app RPS metrics from ClickHouse", result.err);
      return empty;
    }

    // ClickHouse returns per-bucket request/error counts (the query is
    // already dense; WITH FILL pads empty buckets with zeros). totalRequests
    // sums those raw counts; the rps series divides each bucket's count by
    // the bucket length in seconds to get an average requests per second rate.
    const bucketSeconds = query.bucketMs / 1000;
    const perSecond = (count: number) => Math.round((count / bucketSeconds) * 100) / 100;

    let totalRequests = 0;
    const rps: Point[] = result.val.map((row) => {
      totalRequests += row.requests;
      return { time: row.t, requests: perSecond(row.requests), errors: perSecond(row.errors) };
    });

    const output = { range: query.range, totalRequests, rps };

    console.info({ input, app, output });

    return output;
  });
