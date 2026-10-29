import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import type { InstanceRow, InstanceSeries } from "@/lib/metrics/app-metrics";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { HISTORY_INSTANCES_MAX } from "@unkey/clickhouse/src/app-metrics/resources";
import { z } from "zod";
import { currentDeploymentId, deployRole } from "./current-deployment";
import { deploySourceOf } from "./deploy-markers";
import { liveInstancesOf } from "./instances";
import { loadMetricsScope, loadProductionRouting, metricsScopeInput, orThrow } from "./scope";

const instanceSeriesInput = metricsScopeInput.extend({
  openInstance: z.string().max(255).nullable(),
});

export const appMetricsSourceByInstance = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(instanceSeriesInput)
  .query(async ({ ctx, input }): Promise<InstanceSeries> => {
    const scope = await loadMetricsScope(ctx.workspace.id, input);
    const live = await db.query.instances.findMany({
      where: liveInstancesOf(scope),
      columns: { k8sName: true, deploymentId: true },
      with: { deployment: { columns: { status: true, createdAt: true } } },
    });
    const historical = orThrow(
      await clickhouse.appMetrics.resourceHistory({
        ...scope,
        instances: live.map((i) => i.k8sName),
      }),
    );
    const kept = historical.slice(0, HISTORY_INSTANCES_MAX);
    const deploymentIds = [...new Set(kept.map((i) => i.deploymentId))];
    const [deployments, routing] = await Promise.all([
      deploymentIds.length === 0
        ? []
        : db.query.deployments.findMany({
            where: (table, { and, eq, inArray }) =>
              and(
                eq(table.workspaceId, scope.workspaceId),
                eq(table.appId, scope.appId),
                eq(table.environmentId, scope.environmentId),
                inArray(table.id, deploymentIds),
              ),
            columns: {
              id: true,
              status: true,
              gitCommitSha: true,
              gitCommitMessage: true,
              gitCommitAuthorHandle: true,
              gitCommitAuthorAvatarUrl: true,
              imageRequested: true,
            },
          }),
      loadProductionRouting(scope),
    ]);
    const currentId = currentDeploymentId({
      routing,
      live: live.map((i) => ({ id: i.deploymentId, ...i.deployment })),
    });
    const byDeployment = new Map(deployments.map((d) => [d.id, d]));
    const history: InstanceSeries["history"] = {
      truncated: historical.length > HISTORY_INSTANCES_MAX,
      instances: kept.map((i): InstanceRow => {
        const deployment = byDeployment.get(i.deploymentId);
        return {
          instanceId: null,
          region: { name: i.region || "Unknown region" },
          status: "inactive",
          role: deployment ? deployRole(deployment, currentId) : "previous",
          lastExit: null,
          deployment: {
            id: i.deploymentId,
            source: deployment ? deploySourceOf(deployment) : null,
          },
          replica: {
            k8sName: i.instance,
            cpuMillicores: i.cpuMillicores,
            memoryMib: i.memoryMib,
            storageMib: i.storageMib,
          },
        };
      }),
    };
    const instances = [
      ...new Set([
        ...live.map((i) => i.k8sName),
        ...kept.map((i) => i.instance),
        ...(input.openInstance ? [input.openInstance] : []),
      ]),
    ];
    if (instances.length === 0) {
      return { grid: scope.grid, instances: [], history };
    }
    const points = orThrow(
      await clickhouse.appMetrics.resourcesByInstance({ ...scope, instances }),
    );
    const byInstance = Map.groupBy(points, (point) => point.instance);
    return {
      grid: scope.grid,
      history,
      instances: [...byInstance]
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([instance, points]) => ({
          instance,
          points: points.map(({ instance: _, ...point }) => point),
        })),
    };
  });
