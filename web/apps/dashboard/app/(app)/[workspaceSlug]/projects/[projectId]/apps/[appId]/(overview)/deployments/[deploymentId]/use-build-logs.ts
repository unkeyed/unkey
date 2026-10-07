import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { queryKeys } from "@/lib/query-keys";
import { trpc } from "@/lib/trpc/client";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { BuildLogEntry } from "@unkey/api/models/components";

// Rendering every entry of a noisy build is slow, and the end of a build is
// where a failure shows, so only the newest entries are kept
export const BUILD_LOG_ENTRIES_SHOWN_MAX = 5000;

const IN_FLIGHT_POLL_MS = 1_000;
// A quiet step, such as a long compile, prints nothing for a while, so each
// empty poll after the first entry doubles the wait up to this bound. Before
// the first entry the build is starting and output is about to arrive
const IN_FLIGHT_POLL_MAX_MS = 5_000;
const PARTIAL_RESULT_INTERVAL_MS = 250;
const BUILD_LOGS_PAGE_ENTRIES_MAX = 500;
const SETTLED_POLL_MS = 2_000;
// ctrl flushes build logs to ClickHouse up to 2s after they are written, so
// the last lines can arrive after the deployment settles. An empty poll this
// long after the deployment settles means the logs are complete. The window
// bounds the polls when the deployment has no logs or a poll fails
const SETTLED_DRAINED_AFTER_MS = 5_000;
const SETTLED_POLL_WINDOW_MS = 30_000;
// ctrl writes one of these when a build step finishes
const BUILD_STEP_FINISHED = /^(DONE \d|CACHED$|ERROR: )/;

type BuildLogs = {
  entries: BuildLogEntry[];
  entriesTotal: number;
  cursor: string | undefined;
  hasMore: boolean;
  emptyPollsInRow: number;
  isDrained: boolean;
};

// A read to the end publishes what it has every PARTIAL_RESULT_INTERVAL_MS, so
// slow pages show progress without a render per page
export function useBuildLogs(
  deployment: Deployment,
  { readsToEnd, isOpen }: { readsToEnd: boolean; isOpen: boolean },
) {
  const queryClient = useQueryClient();
  const trpcUtils = trpc.useUtils();
  const queryKey = queryKeys.deployments.buildLogs(deployment.id);

  return useQuery({
    queryKey,
    queryFn: async (): Promise<BuildLogs> => {
      let logs: BuildLogs = queryClient.getQueryData<BuildLogs>(queryKey) ?? {
        entries: [],
        entriesTotal: 0,
        cursor: undefined,
        hasMore: true,
        emptyPollsInRow: 0,
        isDrained: false,
      };
      const entriesTotalBefore = logs.entriesTotal;
      const pollStartedAt = Date.now();
      let hasFinishedStep = false;
      let publishedAt = Date.now();
      do {
        const page = await getUnkeyClient().deployments.listBuildLogs({
          deploymentId: deployment.id,
          cursor: logs.cursor,
          limit: BUILD_LOGS_PAGE_ENTRIES_MAX,
        });
        logs = {
          entries: [...logs.entries, ...page.data].slice(-BUILD_LOG_ENTRIES_SHOWN_MAX),
          entriesTotal: logs.entriesTotal + page.data.length,
          cursor: page.pagination.cursor ?? logs.cursor,
          hasMore: page.pagination.hasMore,
          emptyPollsInRow: logs.emptyPollsInRow,
          isDrained: logs.isDrained,
        };
        hasFinishedStep ||= page.data.some((entry) => BUILD_STEP_FINISHED.test(entry.message));
        if (logs.hasMore && Date.now() - publishedAt >= PARTIAL_RESULT_INTERVAL_MS) {
          queryClient.setQueryData(queryKey, logs);
          publishedAt = Date.now();
        }
      } while (readsToEnd && logs.hasMore);
      const isInFlight = isDeploymentInFlight(deployment.status);
      if (hasFinishedStep && isInFlight) {
        void trpcUtils.deploy.deployment.steps.invalidate({ deploymentId: deployment.id });
      }
      const isEmptyPoll = logs.entriesTotal === entriesTotalBefore;
      return {
        ...logs,
        emptyPollsInRow: logs.entriesTotal > 0 && isEmptyPoll ? logs.emptyPollsInRow + 1 : 0,
        isDrained:
          !isInFlight &&
          isEmptyPoll &&
          pollStartedAt - settledAt(deployment) >= SETTLED_DRAINED_AFTER_MS,
      };
    },
    refetchInterval: (data, query) => {
      if (!isOpen || (data?.hasMore && query.state.status !== "error")) {
        return false;
      }
      return pollInterval(deployment, data);
    },
  });
}

function pollInterval(deployment: Deployment, logs: BuildLogs | undefined): number | false {
  if (isDeploymentInFlight(deployment.status)) {
    return Math.min(IN_FLIGHT_POLL_MS * 2 ** (logs?.emptyPollsInRow ?? 0), IN_FLIGHT_POLL_MAX_MS);
  }
  if (logs?.isDrained) {
    return false;
  }
  return Date.now() - settledAt(deployment) < SETTLED_POLL_WINDOW_MS ? SETTLED_POLL_MS : false;
}

function settledAt(deployment: Deployment): number {
  return deployment.updatedAt ?? deployment.createdAt;
}
