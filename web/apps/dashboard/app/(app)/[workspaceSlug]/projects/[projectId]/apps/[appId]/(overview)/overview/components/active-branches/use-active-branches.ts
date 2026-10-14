import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
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

  const branches = useMemo(
    () => (query.data?.pages ?? []).flatMap((page) => page.branches),
    [query.data],
  );

  return {
    branches,
    isLoading: query.isInitialLoading,
    isError: query.isError,
    refetch: query.refetch,
    hasNextPage: query.hasNextPage ?? false,
    isFetchingNextPage: query.isFetchingNextPage,
    fetchNextPage: query.fetchNextPage,
  };
}
