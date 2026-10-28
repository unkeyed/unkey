import { and, db, desc, eq, gte, isNull, lt, or, schema, sql } from "@/lib/db";
import {
  RANGE_PRESETS,
  clampedGrid,
  environmentRefSchema,
  rollupsServe,
} from "@/lib/metrics/app-metrics";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const environmentScopeInput = z.object({
  appId: z.string(),
  environment: environmentRefSchema,
});

export const metricsScopeInput = environmentScopeInput.extend({
  range: z.enum(RANGE_PRESETS),
  endMs: z.number().int().nonnegative(),
});

export type EnvironmentScope = Awaited<ReturnType<typeof loadEnvironmentScope>>;
export type MetricsScope = Awaited<ReturnType<typeof loadMetricsScope>>;

export async function loadEnvironmentScope(
  workspaceId: string,
  input: z.infer<typeof environmentScopeInput>,
) {
  const ref = input.environment;
  const environment = await db.query.environments.findFirst({
    where: (table, { and, eq }) =>
      and(
        "id" in ref ? eq(table.id, ref.id) : and(eq(table.kind, ref.kind), isNull(table.deletedAt)),
        eq(table.appId, input.appId),
        eq(table.workspaceId, workspaceId),
      ),
    columns: { id: true, projectId: true, kind: true },
    orderBy: (table, { asc }) => [asc(table.createdAt)],
  });
  if (!environment) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
  }
  return {
    workspaceId,
    projectId: environment.projectId,
    appId: input.appId,
    environmentId: environment.id,
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

const MAX_MARKER_DEPLOYMENTS = 100;

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

  return (
    db
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
      )
      // Newest first, so a busy range drops its oldest deploys rather than the latest
      .orderBy(desc(d.firstReadyAt), desc(d.id))
      .limit(MAX_MARKER_DEPLOYMENTS)
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
