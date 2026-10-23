import { and, db, desc, eq, gte, inArray, lt, ne, or, schema, sql } from "@/lib/db";
import { RANGE_PRESETS, clampedGrid, rollupsServe } from "@/lib/metrics/app-metrics";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const environmentScopeInput = z.object({
  appId: z.string(),
  environmentId: z.string(),
});

export const metricsScopeInput = environmentScopeInput.extend({
  range: z.enum(RANGE_PRESETS),
  endMs: z.number().int().nonnegative(),
});

// A stopped pod has no MySQL row any more, so an open replica sheet names it to
// keep its last usage in the series.
export const instanceSeriesInput = metricsScopeInput.extend({
  stoppedInstance: z.string().nullable(),
});

export type EnvironmentScope = Awaited<ReturnType<typeof loadEnvironmentScope>>;
export type MetricsScope = Awaited<ReturnType<typeof loadMetricsScope>>;

export async function loadEnvironmentScope(
  workspaceId: string,
  input: z.infer<typeof environmentScopeInput>,
) {
  const environment = await db.query.environments.findFirst({
    where: (table, { and, eq }) =>
      and(
        eq(table.id, input.environmentId),
        eq(table.appId, input.appId),
        eq(table.workspaceId, workspaceId),
      ),
    columns: { projectId: true, kind: true },
  });
  if (!environment) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
  }
  return {
    workspaceId,
    projectId: environment.projectId,
    appId: input.appId,
    environmentId: input.environmentId,
    environmentKind: environment.kind,
  };
}

export async function loadMetricsScope(
  workspaceId: string,
  input: z.infer<typeof metricsScopeInput>,
) {
  const environment = await loadEnvironmentScope(workspaceId, input);
  const nowMs = Date.now();
  const grid = clampedGrid(input.range, input.endMs, nowMs);
  if (!rollupsServe(grid, nowMs)) {
    throw new TRPCError({ code: "BAD_REQUEST", message: "Metrics are not kept that far back" });
  }

  return { ...environment, grid };
}

export type MarkerDeployment = Awaited<ReturnType<typeof loadMarkerDeployments>>[number];

export function loadMarkerDeployments(scope: MetricsScope) {
  const d = schema.deployments;
  const { startMs, endMs } = scope.grid;
  const inEnvironment = and(
    eq(d.workspaceId, scope.workspaceId),
    eq(d.appId, scope.appId),
    eq(d.environmentId, scope.environmentId),
    lt(d.createdAt, endMs),
  );
  const servingAtStart = db
    .select({ id: d.id })
    .from(d)
    .where(and(inEnvironment, lt(d.firstReadyAt, startMs)))
    .orderBy(desc(d.firstReadyAt), desc(d.id))
    .limit(1);

  return db
    .select({
      id: d.id,
      firstReadyAt: d.firstReadyAt,
      gitCommitSha: d.gitCommitSha,
      gitCommitMessage: d.gitCommitMessage,
      gitCommitAuthorHandle: d.gitCommitAuthorHandle,
      gitCommitAuthorAvatarUrl: d.gitCommitAuthorAvatarUrl,
      imageRequested: d.imageRequested,
    })
    .from(d)
    .where(
      and(
        inEnvironment,
        lt(d.firstReadyAt, endMs),
        or(gte(d.firstReadyAt, startMs), eq(d.id, sql`(${servingAtStart})`)),
      ),
    );
}

export function liveInstancesOf(scope: EnvironmentScope) {
  const d = schema.deployments;
  const i = schema.instances;
  const environmentDeployments = db
    .select({ id: d.id })
    .from(d)
    .where(
      and(
        eq(d.workspaceId, scope.workspaceId),
        eq(d.appId, scope.appId),
        eq(d.environmentId, scope.environmentId),
      ),
    );
  return and(
    eq(i.workspaceId, scope.workspaceId),
    eq(i.appId, scope.appId),
    inArray(i.deploymentId, environmentDeployments),
    ne(i.status, "inactive"),
  );
}

export function orThrow<T>(result: { val?: T; err?: Error }): T {
  if (result.err || result.val === undefined) {
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to load app metrics",
      cause: result.err,
    });
  }
  return result.val;
}

export type ProductionRouting = {
  currentDeploymentId: string | null;
  isRolledBack: boolean;
  currentReadyAtMs: number | null;
};

export async function loadProductionRouting(
  scope: EnvironmentScope,
): Promise<ProductionRouting | null> {
  if (scope.environmentKind !== "production") {
    return null;
  }
  const a = schema.apps;
  const d = schema.deployments;
  const [row] = await db
    .select({
      currentDeploymentId: a.currentDeploymentId,
      isRolledBack: a.isRolledBack,
      currentReadyAtMs: d.firstReadyAt,
    })
    .from(a)
    .leftJoin(d, and(eq(d.id, a.currentDeploymentId), eq(d.workspaceId, a.workspaceId)))
    .where(and(eq(a.id, scope.appId), eq(a.workspaceId, scope.workspaceId)))
    .limit(1);
  return {
    currentDeploymentId: row?.currentDeploymentId ?? null,
    isRolledBack: row?.isRolledBack ?? false,
    currentReadyAtMs: row?.currentReadyAtMs ?? null,
  };
}
