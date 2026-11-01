import { useAppScope } from "@/lib/deploy-data";
import { useInfiniteList } from "@/lib/loadable/list";
import { trpc } from "@/lib/trpc/client";

export const ACTIVE_BRANCHES_PAGE_SIZE = 10;

export function useActiveBranches() {
  const { projectId, appId } = useAppScope();

  const query = trpc.deploy.deployment.listActiveBranches.useInfiniteQuery(
    { projectId, appId, limit: ACTIVE_BRANCHES_PAGE_SIZE },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    },
  );

  return useInfiniteList(query, (page) => page.branches);
}
