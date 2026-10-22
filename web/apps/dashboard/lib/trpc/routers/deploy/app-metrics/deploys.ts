import type { DeployMarker } from "@/lib/metrics/app-metrics";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { toDeployMarkers } from "./deploy-markers";
import {
  loadMarkerDeployments,
  loadMetricsScope,
  loadProductionRouting,
  metricsScopeInput,
} from "./scope";

export const appMetricsDeploys = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(metricsScopeInput)
  .query(async ({ ctx, input }): Promise<DeployMarker[]> => {
    const scope = await loadMetricsScope(ctx.workspace.id, input);
    const [deployments, routing] = await Promise.all([
      loadMarkerDeployments(scope),
      loadProductionRouting(scope),
    ]);
    return toDeployMarkers(deployments, scope.grid, routing);
  });
