import { clickhouse } from "@/lib/clickhouse";
import type { SourceResult } from "@/lib/metrics/app-metrics";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { loadMetricsScope, metricsScopeInput, orThrow } from "./scope";

const sourceProcedure = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(metricsScopeInput);

export const appMetricsTraffic = sourceProcedure.query(
  async ({ ctx, input }): Promise<SourceResult<"traffic">> => {
    const scope = await loadMetricsScope(ctx.workspace.id, input);
    return { grid: scope.grid, points: orThrow(await clickhouse.appMetrics.traffic(scope)) };
  },
);

export const appMetricsResources = sourceProcedure.query(
  async ({ ctx, input }): Promise<SourceResult<"resources">> => {
    const scope = await loadMetricsScope(ctx.workspace.id, input);
    return { grid: scope.grid, points: orThrow(await clickhouse.appMetrics.resources(scope)) };
  },
);
