import { clickhouse } from "@/lib/clickhouse";
import type { InstanceEvent } from "@/lib/metrics/app-metrics";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { resolveK8sNamesToInstanceIds } from "../runtime-logs/utils";
import { EPISODE_WAITING_REASONS, toInstanceEvents } from "./instance-event-rows";
import { loadMetricsScope, metricsScopeInput, orThrow } from "./scope";

const MAX_EVENTS_PER_POD = 300;
const MAX_EVENTS = 5_000;

export const appMetricsInstanceEvents = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(metricsScopeInput)
  .query(async ({ ctx, input }): Promise<InstanceEvent[]> => {
    const scope = await loadMetricsScope(ctx.workspace.id, input);
    const rows = orThrow(
      await clickhouse.instanceEvents.crashTimeline({
        workspaceId: scope.workspaceId,
        projectId: scope.projectId,
        environmentId: scope.environmentId,
        startTime: scope.grid.startMs,
        endTime: scope.grid.endMs,
        waitingReasons: EPISODE_WAITING_REASONS,
        perPodLimit: MAX_EVENTS_PER_POD,
        limit: MAX_EVENTS,
      }),
    );
    const instanceIds = await resolveK8sNamesToInstanceIds(
      rows.map((row) => ({ k8sPodName: row.pod_name, region: row.region })),
      scope.workspaceId,
    );
    return toInstanceEvents(rows, instanceIds);
  });
