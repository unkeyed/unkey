import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import {
  APP_METRICS_GROUPS,
  APP_METRICS_WINDOWS,
  APP_RESOURCE_METRICS,
  resolveAppMetricsRange,
} from "@unkey/clickhouse";
import { z } from "zod";

const scopeInput = z.object({
  appId: z.string(),
  environmentId: z.string(),
  window: z.enum(APP_METRICS_WINDOWS),
  groupBy: z.enum(APP_METRICS_GROUPS).default("none"),
});

type Scope = z.infer<typeof scopeInput>;

// The environment carries the project id and must belong to both the app and
// the workspace, so one lookup authorizes the whole scope.
async function resolveScope(workspaceId: string, input: Scope) {
  const environment = await db.query.environments.findFirst({
    where: (table, { eq, and }) =>
      and(
        eq(table.id, input.environmentId),
        eq(table.appId, input.appId),
        eq(table.workspaceId, workspaceId),
      ),
    columns: { id: true, projectId: true, appId: true },
  });
  if (!environment) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
  }
  const range = resolveAppMetricsRange(input.window, Date.now());
  return {
    range,
    query: {
      workspaceId,
      projectId: environment.projectId,
      appId: environment.appId,
      environmentId: environment.id,
      window: input.window,
      groupBy: input.groupBy,
      startMs: range.startMs,
      endMs: range.endMs,
    },
  };
}

function failed(message: string, cause: unknown): TRPCError {
  return new TRPCError({ code: "INTERNAL_SERVER_ERROR", message, cause });
}

export const getAppResourceTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(scopeInput.extend({ metric: z.enum(APP_RESOURCE_METRICS) }))
  .query(async ({ ctx, input }) => {
    const { range, query } = await resolveScope(ctx.workspace.id, input);
    const result = await clickhouse.appMetrics.resources({ ...query, metric: input.metric });
    if (result.err) {
      throw failed("Failed to fetch resource metrics", result.err);
    }
    return { range, points: result.val };
  });

export const getAppRequestTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(scopeInput)
  .query(async ({ ctx, input }) => {
    const { range, query } = await resolveScope(ctx.workspace.id, input);
    const result = await clickhouse.appMetrics.requests(query);
    if (result.err) {
      throw failed("Failed to fetch request metrics", result.err);
    }
    return { range, points: result.val };
  });

export const getAppLatencyTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(scopeInput)
  .query(async ({ ctx, input }) => {
    const { range, query } = await resolveScope(ctx.workspace.id, input);
    const result = await clickhouse.appMetrics.latency(query);
    if (result.err) {
      throw failed("Failed to fetch latency metrics", result.err);
    }
    return { range, points: result.val };
  });

// Deployments created inside the window, for the deploy markers. Deployments
// that started before the window but still serve traffic are not markers;
// they show up as series when grouping by deployment instead.
export const getAppDeploymentMarkers = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(scopeInput.omit({ groupBy: true }))
  .query(async ({ ctx, input }) => {
    const { range, query } = await resolveScope(ctx.workspace.id, { ...input, groupBy: "none" });
    const rows = await db.query.deployments.findMany({
      where: (table, { eq, and, gte, lt }) =>
        and(
          eq(table.workspaceId, ctx.workspace.id),
          eq(table.appId, query.appId),
          eq(table.environmentId, query.environmentId),
          gte(table.createdAt, range.startMs),
          lt(table.createdAt, range.endMs),
        ),
      columns: {
        id: true,
        createdAt: true,
        status: true,
        gitCommitSha: true,
        gitCommitMessage: true,
        gitBranch: true,
      },
      orderBy: (table, { asc }) => asc(table.createdAt),
    });
    return { range, markers: rows };
  });
