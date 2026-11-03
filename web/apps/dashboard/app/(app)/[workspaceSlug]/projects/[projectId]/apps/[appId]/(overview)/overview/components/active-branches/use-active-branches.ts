import { RECENT_DEPLOYMENTS_LIMIT, useAppScope, useRecentDeployments } from "@/lib/deploy-data";
import { useInfiniteList } from "@/lib/loadable/list";
import { trpc } from "@/lib/trpc/client";
import { useEffect, useRef } from "react";

export const ACTIVE_BRANCHES_PAGE_SIZE = 10;

export function useActiveBranches() {
  const { projectId, appId } = useAppScope();
  const trpcUtils = trpc.useUtils();

  const query = trpc.deploy.deployment.listActiveBranches.useInfiniteQuery(
    { projectId, appId, limit: ACTIVE_BRANCHES_PAGE_SIZE },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    },
  );

  // Active branches only move when a deployment does, and a new or moving
  // deployment shows first among the newest
  const { data: recent = [] } = useRecentDeployments({ limit: RECENT_DEPLOYMENTS_LIMIT });
  const recentStates = recent.map((d) => `${d.id}:${d.status}`).join(",");
  const previousStates = useRef(recentStates);
  useEffect(() => {
    const previous = previousStates.current;
    previousStates.current = recentStates;
    if (previous !== "" && previous !== recentStates) {
      trpcUtils.deploy.deployment.listActiveBranches.invalidate();
    }
  }, [recentStates, trpcUtils]);

  return useInfiniteList(query, (page) => page.branches);
}
