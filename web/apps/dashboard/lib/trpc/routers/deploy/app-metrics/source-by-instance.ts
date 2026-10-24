import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import type { InstanceSeries } from "@/lib/metrics/app-metrics";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { liveInstancesOf } from "./instances";
import { loadMetricsScope, metricsScopeInput, orThrow } from "./scope";

const instanceSeriesInput = metricsScopeInput.extend({
  openInstance: z.string().max(255).nullable(),
});

// Live pods resolve here; the client names only the open sheet's pod, which may have stopped
export const appMetricsSourceByInstance = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(instanceSeriesInput)
  .query(async ({ ctx, input }): Promise<InstanceSeries> => {
    const scope = await loadMetricsScope(ctx.workspace.id, input);
    const live = await db.query.instances.findMany({
      where: liveInstancesOf(scope),
      columns: { k8sName: true },
    });
    const instances = [
      ...new Set([
        ...live.map((i) => i.k8sName),
        ...(input.openInstance ? [input.openInstance] : []),
      ]),
    ];
    if (instances.length === 0) {
      return { grid: scope.grid, instances: [] };
    }
    const points = orThrow(
      await clickhouse.appMetrics.resourcesByInstance({ ...scope, instances }),
    );
    const byInstance = Map.groupBy(points, (point) => point.instance);
    return {
      grid: scope.grid,
      instances: [...byInstance]
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([instance, points]) => ({
          instance,
          points: points.map(({ instance: _, ...point }) => point),
        })),
    };
  });
