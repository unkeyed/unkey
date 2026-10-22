import { clickhouse } from "@/lib/clickhouse";
import type { InstanceSeries } from "@/lib/metrics/app-metrics";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { loadMetricsScope, metricsScopeInput, orThrow } from "./scope";

export const appMetricsSourceByInstance = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(metricsScopeInput)
  .query(async ({ ctx, input }): Promise<InstanceSeries> => {
    const scope = await loadMetricsScope(ctx.workspace.id, input);
    const points = orThrow(await clickhouse.appMetrics.resourcesByInstance(scope));
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
