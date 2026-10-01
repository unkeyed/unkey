import type { Deployment } from "@/lib/collections/deploy/deployments";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { BuildLogEntry } from "@unkey/api/models/components";
import { buildPollInterval } from "./use-build-steps";

// Rendering every entry of a noisy step is slow, and the end of a step is
// where a failure shows, so only the newest entries are kept
export const BUILD_STEP_LOG_ENTRIES_SHOWN_MAX = 500;

const NEXT_PAGE_POLL_MS = 1;

type BuildStepLogs = {
  entries: BuildLogEntry[];
  entriesTotal: number;
  cursor: string | undefined;
  hasMore: boolean;
};

export function useBuildStepLogs(deployment: Deployment, stepId: string) {
  const queryClient = useQueryClient();
  const queryKey = queryKeys.deployments.buildLogs(deployment.id, stepId);

  return useQuery({
    queryKey,
    // Each fetch reads one page after the cursor of the previous fetch and
    // appends it, so the entries render while the rest is still loading
    queryFn: async (): Promise<BuildStepLogs> => {
      const previous = queryClient.getQueryData<BuildStepLogs>(queryKey);
      const page = await getUnkeyClient().deployments.getBuildLogs({
        deploymentId: deployment.id,
        stepId,
        cursor: previous?.cursor,
      });
      return {
        entries: [...(previous?.entries ?? []), ...page.data].slice(
          -BUILD_STEP_LOG_ENTRIES_SHOWN_MAX,
        ),
        entriesTotal: (previous?.entriesTotal ?? 0) + page.data.length,
        cursor: page.pagination.cursor ?? previous?.cursor,
        hasMore: page.pagination.hasMore,
      };
    },
    refetchInterval: (data) => (data?.hasMore ? NEXT_PAGE_POLL_MS : buildPollInterval(deployment)),
  });
}
