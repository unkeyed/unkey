import { clickhouse } from "@/lib/clickhouse";
import type { InstanceSeries } from "@/lib/metrics/app-metrics";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { loadMetricsScope, metricsScopeInput, orThrow } from "./scope";

const instanceSeriesInput = metricsScopeInput.extend({
  instances: z.array(z.string().max(255)).max(500),
});

export const appMetricsSourceByInstance = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(instanceSeriesInput)
  .query(async ({ ctx, input }): Promise<InstanceSeries> => {
    const scope = await loadMetricsScope(ctx.workspace.id, input);
    const instances = [...new Set(input.instances)];
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
