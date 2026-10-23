import { clickhouse } from "@/lib/clickhouse";
import { db, schema } from "@/lib/db";
import type { InstanceSeries } from "@/lib/metrics/app-metrics";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { instanceSeriesInput, liveInstancesOf, loadMetricsScope, orThrow } from "./scope";

export const appMetricsSourceByInstance = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(instanceSeriesInput)
  .query(async ({ ctx, input }): Promise<InstanceSeries> => {
    const scope = await loadMetricsScope(ctx.workspace.id, input);
    const live = await db
      .select({ k8sName: schema.instances.k8sName })
      .from(schema.instances)
      .where(liveInstancesOf(scope));
    const instances = live.map((i) => i.k8sName);
    if (input.stoppedInstance !== null) {
      instances.push(input.stoppedInstance);
    }
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
