import { useInfiniteList } from "@/lib/loadable/list";
import { trpc } from "@/lib/trpc/client";
import { useAppId, useProjectData } from "../../../data-provider";

export const ACTIVE_BRANCHES_PAGE_SIZE = 10;

export function useActiveBranches() {
  const { projectId } = useProjectData();
  const appId = useAppId();

  const query = trpc.deploy.deployment.listActiveBranches.useInfiniteQuery(
    { projectId, appId, limit: ACTIVE_BRANCHES_PAGE_SIZE },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    },
  );

  return useInfiniteList(query, (page) => page.branches);
}
